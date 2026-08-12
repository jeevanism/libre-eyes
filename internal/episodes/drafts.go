package episodes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/jeevanism/visionopus/internal/auth"
)

const (
	PermissionDraftCreate  = "event_draft.create"
	PermissionDraftRead    = "event_draft.read"
	PermissionDraftUpdate  = "event_draft.update"
	PermissionDraftAbandon = "event_draft.abandon"

	autosaveDraftLifetime           = 24 * time.Hour
	manualDraftLifetime             = 30 * 24 * time.Hour
	prescriptionManualDraftLifetime = 7 * 24 * time.Hour
	maximumDraftDepth               = 16
	maximumDraftKeys                = 2000
)

type DraftIntent string

const (
	DraftIntentCreate DraftIntent = "create"
	DraftIntentUpdate DraftIntent = "update"
)

type DraftMode string

const (
	DraftModeAutosave DraftMode = "autosave"
	DraftModeManual   DraftMode = "manual"
)

// DraftPayloadRegistry validates a module-owned temporary payload before it is persisted.
type DraftPayloadRegistry interface {
	Validate(ctx context.Context, eventTypeCode string, intent DraftIntent, schemaVersion int64, payload json.RawMessage) error
}

// RejectingDraftRegistry is safe by default until a clinical module registers a schema.
type RejectingDraftRegistry struct{}

func (RejectingDraftRegistry) Validate(context.Context, string, DraftIntent, int64, json.RawMessage) error {
	return ErrInvalidRequest
}

type EventDraft struct {
	ID                  string
	EpisodeID           string
	EventTypeCode       string
	TargetEventID       *string
	Intent              DraftIntent
	Mode                DraftMode
	SchemaVersion       int64
	Payload             json.RawMessage
	Version             int64
	ExpiresAt           time.Time
	NewerCommittedEdits bool
}

type DraftCreateRequest struct {
	EventTypeCode string
	TargetEventID *string
	Intent        DraftIntent
	Mode          DraftMode
	SchemaVersion int64
	Payload       json.RawMessage
}

type DraftUpdateRequest struct {
	ExpectedVersion int64
	SchemaVersion   int64
	Payload         json.RawMessage
}

func (s *Service) CreateDraft(ctx context.Context, authorization Authorization, episodeID string, request DraftCreateRequest) (EventDraft, error) {
	if request.Intent == "" {
		request.Intent = DraftIntentCreate
	}
	if !s.validAuthorization(authorization, PermissionDraftCreate) || !validPublicID(episodeID) || !validDraftCreate(request) || validateDraftPayload(request.Payload) != nil {
		return EventDraft{}, ErrInvalidRequest
	}
	if err := s.validateDraftSchema(ctx, request.EventTypeCode, request.Intent, request.SchemaVersion, request.Payload); err != nil {
		return EventDraft{}, err
	}
	now := s.now().UTC()
	publicID, err := newUUIDv4()
	if err != nil {
		return EventDraft{}, fmt.Errorf("generate draft id: %w", err)
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return EventDraft{}, fmt.Errorf("begin draft create: %w", err)
	}
	defer rollback(ctx, tx)
	episode, err := loadEpisode(ctx, tx, authorization.principal, episodeID, true)
	if err != nil {
		return EventDraft{}, err
	}
	if episode.Status == StatusClosed {
		return EventDraft{}, ErrInvalidTransition
	}
	var targetID *int64
	if request.Intent == DraftIntentUpdate {
		if request.TargetEventID == nil {
			return EventDraft{}, ErrInvalidRequest
		}
		var value int64
		err = tx.QueryRow(ctx, `SELECT e.id FROM events e JOIN episodes ep ON ep.id = e.episode_id WHERE e.public_id = $1 AND ep.public_id = $2 AND ep.institution_id = $3 AND ep.site_id = $4 AND ep.firm_id = $5 AND ep.deleted_at IS NULL AND e.institution_id = $3 AND e.site_id = $4 AND e.firm_id = $5 AND e.status = 'current' AND e.deleted_at IS NULL FOR KEY SHARE OF e`, *request.TargetEventID, episodeID, authorization.principal.InstitutionID, authorization.principal.SiteID, authorization.principal.FirmID).Scan(&value)
		if errors.Is(err, pgx.ErrNoRows) {
			return EventDraft{}, auth.ErrForbidden
		}
		if err != nil {
			return EventDraft{}, fmt.Errorf("resolve draft target: %w", err)
		}
		targetID = &value
	}
	expiresAt := now.Add(draftLifetime(request.EventTypeCode, request.Mode))
	var internalTargetID *int64
	var draft EventDraft
	err = tx.QueryRow(ctx, `
		INSERT INTO event_drafts (public_id, owner_user_id, episode_id, patient_id, institution_id, site_id, firm_id, event_type_code, target_event_id, intent, mode, schema_version, payload, expires_at, created_at, updated_at)
		SELECT $1,$2,e.id,e.patient_id,e.institution_id,e.site_id,e.firm_id,$3,$4,$5::event_draft_intent,$6::event_draft_mode,$7,$8::jsonb,$9,$10,$10
		FROM episodes e WHERE e.public_id = $11 AND e.institution_id = $12 AND e.site_id = $13 AND e.firm_id = $14 AND e.deleted_at IS NULL
		RETURNING public_id::text, event_type_code, target_event_id, intent::text, mode::text, schema_version, payload, version, expires_at`,
		publicID, authorization.principal.UserID, request.EventTypeCode, targetID, request.Intent, request.Mode, request.SchemaVersion, request.Payload, expiresAt, now, episodeID, authorization.principal.InstitutionID, authorization.principal.SiteID, authorization.principal.FirmID,
	).Scan(&draft.ID, &draft.EventTypeCode, &internalTargetID, &draft.Intent, &draft.Mode, &draft.SchemaVersion, &draft.Payload, &draft.Version, &draft.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return EventDraft{}, auth.ErrForbidden
	}
	if isDraftConflict(err) {
		return EventDraft{}, auth.ErrConflict
	}
	if err != nil {
		return EventDraft{}, fmt.Errorf("insert event draft: %w", err)
	}
	draft.EpisodeID = episode.ID
	draft.TargetEventID = request.TargetEventID
	if err := appendAudit(ctx, tx, authorization, "event_draft.created", "created", episodeID, map[string]any{"draftId": draft.ID, "mode": draft.Mode, "intent": draft.Intent}); err != nil {
		return EventDraft{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return EventDraft{}, fmt.Errorf("commit draft create: %w", err)
	}
	return draft, nil
}

func (s *Service) GetDraft(ctx context.Context, authorization Authorization, draftID string) (EventDraft, error) {
	if !s.validAuthorization(authorization, PermissionDraftRead) || !validPublicID(draftID) {
		return EventDraft{}, ErrInvalidRequest
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return EventDraft{}, fmt.Errorf("begin draft read: %w", err)
	}
	defer rollback(ctx, tx)
	draft, err := loadDraft(ctx, tx, authorization.principal, draftID, true)
	if err != nil {
		return EventDraft{}, err
	}
	if err := appendAudit(ctx, tx, authorization, "event_draft.recovered", "disclosed", draft.EpisodeID, map[string]any{"draftId": draft.ID}); err != nil {
		return EventDraft{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return EventDraft{}, fmt.Errorf("commit draft read: %w", err)
	}
	return draft, nil
}

func (s *Service) UpdateDraft(ctx context.Context, authorization Authorization, draftID string, request DraftUpdateRequest) (EventDraft, error) {
	if !s.validAuthorization(authorization, PermissionDraftUpdate) || !validPublicID(draftID) || request.ExpectedVersion < 1 || request.SchemaVersion < 1 || validateDraftPayload(request.Payload) != nil {
		return EventDraft{}, ErrInvalidRequest
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return EventDraft{}, fmt.Errorf("begin draft update: %w", err)
	}
	defer rollback(ctx, tx)
	draft, err := loadDraft(ctx, tx, authorization.principal, draftID, true)
	if err != nil {
		return EventDraft{}, err
	}
	if draft.Version != request.ExpectedVersion {
		return EventDraft{}, auth.ErrConflict
	}
	if err := s.validateDraftSchema(ctx, draft.EventTypeCode, draft.Intent, request.SchemaVersion, request.Payload); err != nil {
		return EventDraft{}, err
	}
	now := s.now().UTC()
	expiresAt := now.Add(draftLifetime(draft.EventTypeCode, draft.Mode))
	err = tx.QueryRow(ctx, `
		UPDATE event_drafts SET payload = $1::jsonb, schema_version = $2, version = version + 1, expires_at = $3, updated_at = $4
		WHERE public_id = $5 AND owner_user_id = $6 AND institution_id = $7 AND site_id = $8 AND firm_id = $9
			AND deleted_at IS NULL AND version = $10
		RETURNING payload, version, expires_at`, request.Payload, request.SchemaVersion, expiresAt, now, draftID,
		authorization.principal.UserID, authorization.principal.InstitutionID, authorization.principal.SiteID, authorization.principal.FirmID, request.ExpectedVersion,
	).Scan(&draft.Payload, &draft.Version, &draft.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return EventDraft{}, auth.ErrConflict
	}
	if isDraftConflict(err) {
		return EventDraft{}, auth.ErrConflict
	}
	if err != nil {
		return EventDraft{}, fmt.Errorf("update event draft: %w", err)
	}
	draft.SchemaVersion = request.SchemaVersion
	if err := appendAudit(ctx, tx, authorization, "event_draft.updated", "updated", draft.EpisodeID, map[string]any{"draftId": draft.ID, "version": draft.Version}); err != nil {
		return EventDraft{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return EventDraft{}, fmt.Errorf("commit draft update: %w", err)
	}
	return draft, nil
}

func draftLifetime(eventTypeCode string, mode DraftMode) time.Duration {
	if mode == DraftModeAutosave {
		return autosaveDraftLifetime
	}
	if eventTypeCode == "ophthalmology.prescription_demo" {
		return prescriptionManualDraftLifetime
	}
	return manualDraftLifetime
}

func (s *Service) AbandonDraft(ctx context.Context, authorization Authorization, draftID string, expectedVersion int64) error {
	if !s.validAuthorization(authorization, PermissionDraftAbandon) || !validPublicID(draftID) || expectedVersion < 1 {
		return ErrInvalidRequest
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return fmt.Errorf("begin draft abandonment: %w", err)
	}
	defer rollback(ctx, tx)
	draft, err := loadDraft(ctx, tx, authorization.principal, draftID, true)
	if err != nil {
		return err
	}
	if draft.Version != expectedVersion {
		return auth.ErrConflict
	}
	now := s.now().UTC()
	result, err := tx.Exec(ctx, `UPDATE event_drafts SET deleted_at = $1, deleted_by_user_id = $2, disposal_reason = 'abandoned', updated_at = $1 WHERE public_id = $3 AND owner_user_id = $2 AND institution_id = $4 AND site_id = $5 AND firm_id = $6 AND deleted_at IS NULL AND version = $7`, now, authorization.principal.UserID, draftID, authorization.principal.InstitutionID, authorization.principal.SiteID, authorization.principal.FirmID, expectedVersion)
	if err != nil {
		if isDraftConflict(err) {
			return auth.ErrConflict
		}
		return fmt.Errorf("abandon event draft: %w", err)
	}
	if result.RowsAffected() != 1 {
		return auth.ErrConflict
	}
	if err := appendAudit(ctx, tx, authorization, "event_draft.abandoned", "abandoned", draft.EpisodeID, map[string]any{"draftId": draft.ID}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit draft abandonment: %w", err)
	}
	return nil
}

// ExpireDueDrafts performs policy-driven disposal without retaining draft payloads in audit data.
// It is intended for a trusted worker, not an HTTP request boundary.
func (s *Service) ExpireDueDrafts(ctx context.Context, systemUserID int64, limit int) (int, error) {
	if systemUserID < 1 || limit < 1 || limit > maximumPageSize {
		return 0, ErrInvalidRequest
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return 0, fmt.Errorf("begin draft expiry: %w", err)
	}
	defer rollback(ctx, tx)
	now := s.now().UTC()
	rows, err := tx.Query(ctx, `
		WITH due AS (
			SELECT d.id, e.public_id::text AS episode_public_id
			FROM event_drafts d JOIN episodes e ON e.id = d.episode_id
			WHERE d.deleted_at IS NULL AND d.expires_at <= $1
			ORDER BY d.expires_at, d.id
			FOR UPDATE OF d SKIP LOCKED
			LIMIT $3
		)
		UPDATE event_drafts d
		SET deleted_at = $1, deleted_by_user_id = $2, disposal_reason = 'expired', updated_at = $1
		FROM due
		WHERE d.id = due.id AND d.deleted_at IS NULL
		RETURNING d.public_id::text, due.episode_public_id, d.institution_id, d.site_id, d.firm_id`, now, systemUserID, limit)
	if err != nil {
		return 0, fmt.Errorf("expire due drafts: %w", err)
	}
	type expiredDraft struct {
		id, episodeID                 string
		institutionID, siteID, firmID int64
	}
	expired := make([]expiredDraft, 0, limit)
	for rows.Next() {
		var draft expiredDraft
		if err := rows.Scan(&draft.id, &draft.episodeID, &draft.institutionID, &draft.siteID, &draft.firmID); err != nil {
			return 0, fmt.Errorf("scan expired draft: %w", err)
		}
		expired = append(expired, draft)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate expired drafts: %w", err)
	}
	rows.Close()
	for _, draft := range expired {
		attributes, err := json.Marshal(map[string]any{"draftId": draft.id, "episodeId": draft.episodeID, "policy": "inactivity_expiry"})
		if err != nil {
			return 0, fmt.Errorf("encode draft expiry audit: %w", err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO audit_events (event_type, actor_user_id, institution_id, site_id, firm_id, outcome, reason_code, correlation_id, source_ip_class, attributes) VALUES ('event_draft.expired',$1,$2,$3,$4,'success','expired','system-draft-expiry','system',$5::jsonb)`, systemUserID, draft.institutionID, draft.siteID, draft.firmID, attributes); err != nil {
			return 0, fmt.Errorf("audit draft expiry: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit draft expiry: %w", err)
	}
	return len(expired), nil
}

func loadDraft(ctx context.Context, tx pgx.Tx, principal auth.OperationPrincipal, publicID string, forUpdate bool) (EventDraft, error) {
	query := `
		SELECT d.public_id::text, e.public_id::text, d.event_type_code, target.public_id::text,
			d.intent::text, d.mode::text, d.schema_version, d.payload, d.version, d.expires_at
		FROM event_drafts d
		JOIN episodes e ON e.id = d.episode_id AND e.deleted_at IS NULL
		LEFT JOIN events target ON target.id = d.target_event_id
		WHERE d.public_id = $1 AND d.owner_user_id = $2 AND d.institution_id = $3 AND d.site_id = $4 AND d.firm_id = $5
			AND d.deleted_at IS NULL AND d.expires_at > now()`
	if forUpdate {
		query += " FOR UPDATE OF d"
	}
	var draft EventDraft
	err := tx.QueryRow(ctx, query, publicID, principal.UserID, principal.InstitutionID, principal.SiteID, principal.FirmID).Scan(&draft.ID, &draft.EpisodeID, &draft.EventTypeCode, &draft.TargetEventID, &draft.Intent, &draft.Mode, &draft.SchemaVersion, &draft.Payload, &draft.Version, &draft.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return EventDraft{}, auth.ErrForbidden
	}
	if err != nil {
		return EventDraft{}, fmt.Errorf("load event draft: %w", err)
	}
	if draft.EventTypeCode == "ophthalmology.visual_acuity" {
		return draft, nil
	}
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM events newer
			JOIN event_drafts d ON d.public_id = $1
			WHERE newer.episode_id = d.episode_id
				AND newer.status = 'current'
				AND newer.deleted_at IS NULL
				AND (newer.created_at > d.updated_at OR newer.updated_at > d.updated_at)
				AND (d.intent = 'create' OR newer.id = d.target_event_id)
		)`, publicID).Scan(&draft.NewerCommittedEdits)
	if err != nil {
		return EventDraft{}, fmt.Errorf("load newer draft edits: %w", err)
	}
	return draft, nil
}

func (s *Service) validateDraftSchema(ctx context.Context, eventTypeCode string, intent DraftIntent, schemaVersion int64, payload json.RawMessage) error {
	err := s.drafts.Validate(ctx, eventTypeCode, intent, schemaVersion, payload)
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrInvalidRequest) {
		return ErrInvalidRequest
	}
	return fmt.Errorf("validate draft schema: %w", ErrUnavailable)
}

func validDraftCreate(request DraftCreateRequest) bool {
	if strings.TrimSpace(request.EventTypeCode) == "" || len(request.EventTypeCode) > 100 || request.SchemaVersion < 1 || (request.Mode != DraftModeAutosave && request.Mode != DraftModeManual) {
		return false
	}
	return (request.Intent == DraftIntentCreate && request.TargetEventID == nil) || (request.Intent == DraftIntentUpdate && request.TargetEventID != nil && validPublicID(*request.TargetEventID))
}

func validateDraftPayload(payload json.RawMessage) error {
	if len(payload) == 0 || len(payload) > 64*1024 {
		return ErrInvalidRequest
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return ErrInvalidRequest
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return ErrInvalidRequest
	}
	if _, ok := value.(map[string]any); !ok {
		return ErrInvalidRequest
	}
	keys := 0
	if err := walkDraftPayload(value, 1, &keys); err != nil {
		return err
	}
	return nil
}

func isDraftConflict(err error) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && (postgresError.Code == "23505" || postgresError.Code == "40001")
}

func walkDraftPayload(value any, depth int, keys *int) error {
	if depth > maximumDraftDepth {
		return ErrInvalidRequest
	}
	switch value := value.(type) {
	case map[string]any:
		*keys += len(value)
		if *keys > maximumDraftKeys {
			return ErrInvalidRequest
		}
		for _, child := range value {
			if err := walkDraftPayload(child, depth+1, keys); err != nil {
				return err
			}
		}
	case []any:
		*keys += len(value)
		if *keys > maximumDraftKeys {
			return ErrInvalidRequest
		}
		for _, child := range value {
			if err := walkDraftPayload(child, depth+1, keys); err != nil {
				return err
			}
		}
	}
	return nil
}
