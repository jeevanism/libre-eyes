package episodes

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jeevanism/libre-eyes/internal/auth"
)

type operationAuthorizer interface {
	AuthorizeOperation(context.Context, auth.OperationAuthorizationRequest) (auth.OperationPrincipal, error)
}

// Authorization proves that this service authorized one named operation.
// Its principal is intentionally private so callers cannot construct one.
type Authorization struct {
	principal  auth.OperationPrincipal
	permission string
	metadata   auth.RequestMetadata
	service    *Service
}

// Service owns authorized episode lifecycle operations and their audit records.
type Service struct {
	pool       *pgxpool.Pool
	authorizer operationAuthorizer
	drafts     DraftPayloadRegistry
	cursors    *cursorStore
	now        func() time.Time
}

// NewService constructs an episode service with its required dependencies.
func NewService(pool *pgxpool.Pool, authorizer operationAuthorizer) (*Service, error) {
	return NewServiceWithDraftRegistry(pool, authorizer, RejectingDraftRegistry{})
}

// NewServiceWithDraftRegistry constructs the episode service with a module-owned draft validator.
func NewServiceWithDraftRegistry(pool *pgxpool.Pool, authorizer operationAuthorizer, drafts DraftPayloadRegistry) (*Service, error) {
	if pool == nil || authorizer == nil || drafts == nil {
		return nil, errors.New("episode database and authorizer are required")
	}
	service := &Service{pool: pool, authorizer: authorizer, drafts: drafts, cursors: newCursorStore(), now: time.Now}
	service.cursors.now = func() time.Time { return service.now() }
	return service, nil
}

// Authorize validates session, CSRF, current context, and the operation permission.
func (s *Service) Authorize(ctx context.Context, token, csrf, permission string, metadata auth.RequestMetadata) (Authorization, error) {
	deniedEvent, ok := deniedEventFor(permission)
	if !ok {
		return Authorization{}, ErrInvalidRequest
	}
	capability := ""
	if strings.HasPrefix(permission, "event_draft.") {
		capability = "examination"
	}
	principal, err := s.authorizer.AuthorizeOperation(ctx, auth.OperationAuthorizationRequest{
		Token: token, CSRFToken: csrf, Permission: permission, Capability: capability, DeniedEventType: deniedEvent, Metadata: metadata,
	})
	if err != nil {
		return Authorization{}, err
	}
	return Authorization{principal: principal, permission: permission, metadata: metadata, service: s}, nil
}

// Create creates an open or active episode in the authenticated clinical context.
func (s *Service) Create(ctx context.Context, authorization Authorization, patientID string, request CreateRequest) (Episode, error) {
	if !s.validAuthorization(authorization, permissionCreate) || !validPublicID(patientID) || (request.Status != StatusOpen && request.Status != StatusActive) {
		return Episode{}, ErrInvalidRequest
	}
	publicID, err := newUUIDv4()
	if err != nil {
		return Episode{}, fmt.Errorf("generate episode id: %w", err)
	}
	now := s.now().UTC()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return Episode{}, fmt.Errorf("begin episode create: %w", err)
	}
	defer rollback(ctx, tx)

	var patientPK, patientInstitutionID int64
	err = tx.QueryRow(ctx, `
		SELECT p.id, pi.id
		FROM patients p
		JOIN patient_institutions pi ON pi.patient_id = p.id
			AND pi.institution_id = $1 AND pi.active
		WHERE p.public_id = $2 AND p.active
		FOR KEY SHARE`, authorization.principal.InstitutionID, patientID).Scan(&patientPK, &patientInstitutionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Episode{}, auth.ErrForbidden
	}
	if err != nil {
		return Episode{}, fmt.Errorf("resolve episode patient: %w", err)
	}

	var episodeID int64
	var episode Episode
	err = tx.QueryRow(ctx, `
		INSERT INTO episodes (
			public_id, patient_id, institution_id, patient_institution_id, site_id, firm_id,
			status, started_at, created_by_user_id, updated_by_user_id, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$9,$8,$8)
		RETURNING id, public_id::text, status, started_at, ended_at, support_services, change_tracker, version`,
		publicID, patientPK, authorization.principal.InstitutionID, patientInstitutionID,
		authorization.principal.SiteID, authorization.principal.FirmID, request.Status, now,
		authorization.principal.UserID,
	).Scan(&episodeID, &episode.ID, &episode.Status, &episode.StartedAt, &episode.EndedAt, &episode.SupportServices, &episode.ChangeTracker, &episode.Version)
	if err != nil {
		return Episode{}, fmt.Errorf("insert episode: %w", err)
	}
	episode.PatientID = patientID
	if _, err := tx.Exec(ctx, `INSERT INTO episode_audit_sequences (episode_id) VALUES ($1)`, episodeID); err != nil {
		return Episode{}, fmt.Errorf("initialize episode audit sequence: %w", err)
	}
	if err := appendAudit(ctx, tx, authorization, "episode.created", "created_"+string(episode.Status), episode.ID, map[string]any{"status": episode.Status, "version": episode.Version}); err != nil {
		return Episode{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Episode{}, fmt.Errorf("commit episode create: %w", err)
	}
	return episode, nil
}

// List returns a bounded, opaque-cursor timeline page in ordinary clinical scope.
func (s *Service) List(ctx context.Context, authorization Authorization, request ListRequest) (EpisodePage, error) {
	if !s.validAuthorization(authorization, permissionRead) || !validPublicID(request.PatientID) {
		return EpisodePage{}, ErrInvalidRequest
	}
	limit := request.Limit
	if limit == 0 {
		limit = defaultPageSize
	}
	if limit < 1 || limit > maximumPageSize {
		return EpisodePage{}, ErrInvalidRequest
	}
	binding := cursorEntry{
		userID: authorization.principal.UserID, sessionID: authorization.principal.SessionID,
		institutionID: authorization.principal.InstitutionID, siteID: authorization.principal.SiteID,
		firmID: authorization.principal.FirmID, contextVersion: authorization.principal.ContextVersion,
		patientID: sha256.Sum256([]byte(request.PatientID)), limit: limit,
	}
	var boundary *pageBoundary
	if request.Cursor != "" {
		value, err := s.cursors.get(request.Cursor, binding)
		if err != nil {
			return EpisodePage{}, err
		}
		boundary = &value
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return EpisodePage{}, fmt.Errorf("begin episode list: %w", err)
	}
	defer rollback(ctx, tx)
	if err := ensurePatientScope(ctx, tx, authorization.principal, request.PatientID); err != nil {
		return EpisodePage{}, err
	}
	query := `
		SELECT e.id, e.public_id::text, p.public_id::text, e.status, e.started_at, e.ended_at,
			e.support_services, e.change_tracker, e.version
		FROM episodes e
		JOIN patients p ON p.id = e.patient_id
		WHERE p.public_id = $1 AND e.institution_id = $2 AND e.site_id = $3 AND e.firm_id = $4
			AND e.deleted_at IS NULL`
	args := []any{request.PatientID, authorization.principal.InstitutionID, authorization.principal.SiteID, authorization.principal.FirmID}
	if boundary != nil {
		if boundary.startedAt == nil {
			query += " AND e.started_at IS NULL AND e.id < $5"
			args = append(args, boundary.internalID)
		} else {
			query += " AND (e.started_at IS NULL OR e.started_at < $5 OR (e.started_at = $5 AND e.id < $6))"
			args = append(args, *boundary.startedAt, boundary.internalID)
		}
	}
	query += " ORDER BY e.started_at DESC NULLS LAST, e.id DESC LIMIT $" + fmt.Sprint(len(args)+1)
	args = append(args, limit+1)
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return EpisodePage{}, fmt.Errorf("list episodes: %w", err)
	}
	defer rows.Close()
	episodes := make([]Episode, 0, limit)
	boundaries := make([]pageBoundary, 0, limit+1)
	for rows.Next() {
		var internalID int64
		var episode Episode
		if err := rows.Scan(&internalID, &episode.ID, &episode.PatientID, &episode.Status, &episode.StartedAt, &episode.EndedAt, &episode.SupportServices, &episode.ChangeTracker, &episode.Version); err != nil {
			return EpisodePage{}, fmt.Errorf("scan episode: %w", err)
		}
		episodes = append(episodes, episode)
		boundaries = append(boundaries, pageBoundary{startedAt: episode.StartedAt, internalID: internalID})
	}
	if err := rows.Err(); err != nil {
		return EpisodePage{}, fmt.Errorf("iterate episodes: %w", err)
	}
	rows.Close()
	page := EpisodePage{Items: episodes}
	if len(episodes) > limit {
		page.Items = episodes[:limit]
		binding.boundary = boundaries[limit-1]
		token, err := s.cursors.put(binding)
		if err != nil {
			return EpisodePage{}, ErrUnavailable
		}
		page.NextCursor = &token
	}
	if err := appendAudit(ctx, tx, authorization, "episode.listed", "disclosed", "", map[string]any{
		"patientId": request.PatientID, "paginated": request.Cursor != "", "resultCount": len(page.Items),
	}); err != nil {
		return EpisodePage{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return EpisodePage{}, fmt.Errorf("commit episode list: %w", err)
	}
	return page, nil
}

// Get returns one lifecycle header in ordinary clinical scope.
func (s *Service) Get(ctx context.Context, authorization Authorization, episodeID string) (Episode, error) {
	if !s.validAuthorization(authorization, permissionRead) || !validPublicID(episodeID) {
		return Episode{}, ErrInvalidRequest
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return Episode{}, fmt.Errorf("begin episode read: %w", err)
	}
	defer rollback(ctx, tx)
	episode, err := loadEpisode(ctx, tx, authorization.principal, episodeID, false)
	if err != nil {
		return Episode{}, err
	}
	if err := appendAudit(ctx, tx, authorization, "episode.read", "disclosed", episode.ID, nil); err != nil {
		return Episode{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Episode{}, fmt.Errorf("commit episode read: %w", err)
	}
	return episode, nil
}

// ListEvents returns current minimum-disclosure event headers for one authorized episode.
func (s *Service) ListEvents(ctx context.Context, authorization Authorization, request EventListRequest) (EventPage, error) {
	if !s.validAuthorization(authorization, permissionRead) || !validPublicID(request.EpisodeID) {
		return EventPage{}, ErrInvalidRequest
	}
	limit := request.Limit
	if limit == 0 {
		limit = defaultPageSize
	}
	if limit < 1 || limit > maximumPageSize {
		return EventPage{}, ErrInvalidRequest
	}
	binding := cursorEntry{
		userID: authorization.principal.UserID, sessionID: authorization.principal.SessionID,
		institutionID: authorization.principal.InstitutionID, siteID: authorization.principal.SiteID,
		firmID: authorization.principal.FirmID, contextVersion: authorization.principal.ContextVersion,
		episodeID: sha256.Sum256([]byte(request.EpisodeID)), limit: limit,
	}
	var boundary *pageBoundary
	if request.Cursor != "" {
		value, err := s.cursors.get(request.Cursor, binding)
		if err != nil {
			return EventPage{}, err
		}
		boundary = &value
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return EventPage{}, fmt.Errorf("begin event list: %w", err)
	}
	defer rollback(ctx, tx)
	if _, err := loadEpisode(ctx, tx, authorization.principal, request.EpisodeID, false); err != nil {
		return EventPage{}, err
	}
	query := `
		SELECT e.id, e.public_id::text, ep.public_id::text, e.event_type_code,
			e.occurred_at, e.status::text, e.version
		FROM events e
		JOIN episodes ep ON ep.id = e.episode_id
		WHERE ep.public_id = $1 AND e.institution_id = $2 AND e.site_id = $3 AND e.firm_id = $4
			AND ep.deleted_at IS NULL AND e.status = 'current' AND e.deleted_at IS NULL`
	args := []any{request.EpisodeID, authorization.principal.InstitutionID, authorization.principal.SiteID, authorization.principal.FirmID}
	if boundary != nil {
		query += " AND (e.occurred_at < $5 OR (e.occurred_at = $5 AND e.id < $6))"
		args = append(args, boundary.occurredAt, boundary.internalID)
	}
	query += " ORDER BY e.occurred_at DESC, e.id DESC LIMIT $" + fmt.Sprint(len(args)+1)
	args = append(args, limit+1)
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return EventPage{}, fmt.Errorf("list event headers: %w", err)
	}
	defer rows.Close()
	events := make([]EventHeader, 0, limit)
	boundaries := make([]pageBoundary, 0, limit+1)
	for rows.Next() {
		var internalID int64
		var event EventHeader
		if err := rows.Scan(&internalID, &event.ID, &event.EpisodeID, &event.EventTypeCode, &event.OccurredAt, &event.Status, &event.Version); err != nil {
			return EventPage{}, fmt.Errorf("scan event header: %w", err)
		}
		events = append(events, event)
		occurredAt := event.OccurredAt
		boundaries = append(boundaries, pageBoundary{occurredAt: &occurredAt, internalID: internalID})
	}
	if err := rows.Err(); err != nil {
		return EventPage{}, fmt.Errorf("iterate event headers: %w", err)
	}
	rows.Close()
	page := EventPage{Items: events}
	if len(events) > limit {
		page.Items = events[:limit]
		binding.boundary = boundaries[limit-1]
		token, err := s.cursors.put(binding)
		if err != nil {
			return EventPage{}, ErrUnavailable
		}
		page.NextCursor = &token
	}
	if err := appendAudit(ctx, tx, authorization, "event.headers_listed", "disclosed", request.EpisodeID, map[string]any{
		"paginated": request.Cursor != "", "resultCount": len(page.Items),
	}); err != nil {
		return EventPage{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return EventPage{}, fmt.Errorf("commit event list: %w", err)
	}
	return page, nil
}

// GetEventHeader returns one current minimum-disclosure event header in ordinary scope.
func (s *Service) GetEventHeader(ctx context.Context, authorization Authorization, eventID string) (EventHeader, error) {
	if !s.validAuthorization(authorization, permissionRead) || !validPublicID(eventID) {
		return EventHeader{}, ErrInvalidRequest
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return EventHeader{}, fmt.Errorf("begin event header read: %w", err)
	}
	defer rollback(ctx, tx)
	event, err := loadEventHeader(ctx, tx, authorization.principal, eventID)
	if err != nil {
		return EventHeader{}, err
	}
	episodeID := ""
	if event.EpisodeID != nil {
		episodeID = *event.EpisodeID
	}
	if err := appendAudit(ctx, tx, authorization, "event.header_read", "disclosed", episodeID, map[string]any{"eventId": event.ID}); err != nil {
		return EventHeader{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return EventHeader{}, fmt.Errorf("commit event header read: %w", err)
	}
	return event, nil
}

// Activate changes an open episode to active.
func (s *Service) Activate(ctx context.Context, authorization Authorization, request LifecycleRequest) (Episode, error) {
	return s.transition(ctx, authorization, request, permissionUpdate, StatusOpen, StatusActive, "episode.activated")
}

// Close changes an active episode to closed and records its end timestamp.
func (s *Service) Close(ctx context.Context, authorization Authorization, request LifecycleRequest) (Episode, error) {
	return s.transition(ctx, authorization, request, permissionUpdate, StatusActive, StatusClosed, "episode.closed")
}

// Reopen changes a closed episode to active. A non-empty reason is mandatory.
func (s *Service) Reopen(ctx context.Context, authorization Authorization, request LifecycleRequest) (Episode, error) {
	if strings.TrimSpace(request.Reason) == "" || len(request.Reason) > 500 {
		return Episode{}, ErrInvalidRequest
	}
	return s.transition(ctx, authorization, request, permissionReopen, StatusClosed, StatusActive, "episode.reopened")
}

func (s *Service) transition(ctx context.Context, authorization Authorization, request LifecycleRequest, permission string, from, to Status, eventType string) (Episode, error) {
	if !s.validAuthorization(authorization, permission) || !validPublicID(request.EpisodeID) || request.ExpectedVersion < 1 {
		return Episode{}, ErrInvalidRequest
	}
	if permission != permissionReopen && request.Reason != "" {
		return Episode{}, ErrInvalidRequest
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return Episode{}, fmt.Errorf("begin episode transition: %w", err)
	}
	defer rollback(ctx, tx)
	episode, err := loadEpisode(ctx, tx, authorization.principal, request.EpisodeID, true)
	if err != nil {
		return Episode{}, err
	}
	if episode.Version != request.ExpectedVersion {
		return Episode{}, auth.ErrConflict
	}
	if episode.Status != from {
		return Episode{}, ErrInvalidTransition
	}
	now := s.now().UTC()
	var endedAt *time.Time
	if to == StatusClosed {
		endedAt = &now
	}
	err = tx.QueryRow(ctx, `
		UPDATE episodes
		SET status = $1, ended_at = $2, version = version + 1,
			updated_by_user_id = $3, updated_at = $4
		WHERE public_id = $5 AND version = $6
		RETURNING public_id::text, status, started_at, ended_at, support_services, change_tracker, version`,
		to, endedAt, authorization.principal.UserID, now, request.EpisodeID, request.ExpectedVersion,
	).Scan(&episode.ID, &episode.Status, &episode.StartedAt, &episode.EndedAt, &episode.SupportServices, &episode.ChangeTracker, &episode.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return Episode{}, auth.ErrConflict
	}
	if err != nil {
		return Episode{}, fmt.Errorf("update episode lifecycle: %w", err)
	}
	attributes := map[string]any{"priorStatus": from, "resultingStatus": to, "version": episode.Version}
	if request.Reason != "" {
		attributes["reason"] = strings.TrimSpace(request.Reason)
	}
	if err := appendAudit(ctx, tx, authorization, eventType, "transitioned", episode.ID, attributes); err != nil {
		return Episode{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Episode{}, fmt.Errorf("commit episode lifecycle: %w", err)
	}
	return episode, nil
}

func ensurePatientScope(ctx context.Context, tx pgx.Tx, principal auth.OperationPrincipal, patientPublicID string) error {
	var visible bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM patients p
			JOIN patient_institutions pi ON pi.patient_id = p.id
				AND pi.institution_id = $1 AND pi.active
			WHERE p.public_id = $2 AND p.active
		)`, principal.InstitutionID, patientPublicID).Scan(&visible)
	if err != nil {
		return fmt.Errorf("validate episode patient scope: %w", err)
	}
	if !visible {
		return auth.ErrForbidden
	}
	return nil
}

func loadEpisode(ctx context.Context, tx pgx.Tx, principal auth.OperationPrincipal, publicID string, forUpdate bool) (Episode, error) {
	query := `
		SELECT e.public_id::text, p.public_id::text, e.status, e.started_at, e.ended_at,
			e.support_services, e.change_tracker, e.version
		FROM episodes e
		JOIN patients p ON p.id = e.patient_id AND p.active
		JOIN patient_institutions pi ON pi.id = e.patient_institution_id AND pi.active
		WHERE e.public_id = $1 AND e.institution_id = $2 AND e.site_id = $3 AND e.firm_id = $4
			AND e.deleted_at IS NULL`
	if forUpdate {
		query += " FOR UPDATE"
	}
	var episode Episode
	err := tx.QueryRow(ctx, query, publicID, principal.InstitutionID, principal.SiteID, principal.FirmID).Scan(
		&episode.ID, &episode.PatientID, &episode.Status, &episode.StartedAt, &episode.EndedAt,
		&episode.SupportServices, &episode.ChangeTracker, &episode.Version,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		// A resource lookup outside ordinary scope must not reveal whether the
		// guessed public ID exists. Authorized missing-resource handling belongs
		// to a future override-aware HTTP boundary.
		return Episode{}, auth.ErrForbidden
	}
	if err != nil {
		return Episode{}, fmt.Errorf("load episode: %w", err)
	}
	return episode, nil
}

func loadEventHeader(ctx context.Context, tx pgx.Tx, principal auth.OperationPrincipal, publicID string) (EventHeader, error) {
	var event EventHeader
	err := tx.QueryRow(ctx, `
		SELECT e.public_id::text, ep.public_id::text, e.event_type_code,
			e.occurred_at, e.status::text, e.version
		FROM events e
		LEFT JOIN episodes ep ON ep.id = e.episode_id
		JOIN patients p ON p.id = e.patient_id AND p.active
		JOIN patient_institutions pi ON pi.id = e.patient_institution_id AND pi.active
		WHERE e.public_id = $1 AND e.institution_id = $2 AND e.site_id = $3 AND e.firm_id = $4
			AND e.status = 'current' AND e.deleted_at IS NULL
			AND (e.is_imported_orphan OR (ep.id IS NOT NULL AND ep.deleted_at IS NULL))`,
		publicID, principal.InstitutionID, principal.SiteID, principal.FirmID,
	).Scan(&event.ID, &event.EpisodeID, &event.EventTypeCode, &event.OccurredAt, &event.Status, &event.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return EventHeader{}, auth.ErrForbidden
	}
	if err != nil {
		return EventHeader{}, fmt.Errorf("load event header: %w", err)
	}
	return event, nil
}

func appendAudit(ctx context.Context, tx pgx.Tx, authorization Authorization, eventType, reasonCode, episodeID string, attributes map[string]any) error {
	if attributes == nil {
		attributes = map[string]any{}
	}
	if episodeID != "" {
		attributes["episodeId"] = episodeID
	}
	encoded, err := json.Marshal(attributes)
	if err != nil {
		return fmt.Errorf("encode episode audit: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (
			event_type, actor_user_id, session_id, institution_id, site_id, firm_id,
			outcome, reason_code, correlation_id, source_ip_class, attributes
		) VALUES ($1,$2,$3,$4,$5,$6,'success',$7,$8,$9,$10::jsonb)`,
		eventType, authorization.principal.UserID, authorization.principal.SessionID,
		authorization.principal.InstitutionID, authorization.principal.SiteID, authorization.principal.FirmID,
		reasonCode, authorization.metadata.CorrelationID, authorization.metadata.SourceIPClass, encoded,
	); err != nil {
		return fmt.Errorf("append episode audit: %w", err)
	}
	return nil
}

func (s *Service) validAuthorization(authorization Authorization, permission string) bool {
	return authorization.service == s && authorization.permission == permission && authorization.principal.UserID > 0
}

func deniedEventFor(permission string) (string, bool) {
	switch permission {
	case permissionRead:
		return "episode.read_denied", true
	case permissionCreate:
		return "episode.create_denied", true
	case permissionUpdate:
		return "episode.update_denied", true
	case permissionReopen:
		return "episode.reopen_denied", true
	case PermissionDraftCreate:
		return "event_draft.create_denied", true
	case PermissionDraftRead:
		return "event_draft.read_denied", true
	case PermissionDraftUpdate:
		return "event_draft.update_denied", true
	case PermissionDraftAbandon:
		return "event_draft.abandon_denied", true
	default:
		return "", false
	}
}

func newUUIDv4() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	return fmt.Sprintf("%s-%s-%s-%s-%s", hex.EncodeToString(value[0:4]), hex.EncodeToString(value[4:6]), hex.EncodeToString(value[6:8]), hex.EncodeToString(value[8:10]), hex.EncodeToString(value[10:16])), nil
}

func validPublicID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for index, character := range value {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if character != '-' {
				return false
			}
			continue
		}
		if !((character >= '0' && character <= '9') || (character >= 'a' && character <= 'f') || (character >= 'A' && character <= 'F')) {
			return false
		}
	}
	return true
}

func rollback(ctx context.Context, tx pgx.Tx) { _ = tx.Rollback(context.WithoutCancel(ctx)) }
