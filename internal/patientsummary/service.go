package patientsummary

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jeevanism/visionopus/internal/auth"
)

const permissionSummaryRead = "patient.summary.read"
const permissionClinicalSummaryRead = "patient.clinical_summary.read"
const permissionBreakGlass = "patient.break_glass"
const permissionBreakGlassRevoke = "patient.break_glass.revoke"

type operationAuthorizer interface {
	AuthorizeOperation(context.Context, auth.OperationAuthorizationRequest) (auth.OperationPrincipal, error)
}

type Service struct {
	pool       *pgxpool.Pool
	authorizer operationAuthorizer
	now        func() time.Time
}

type Authorization struct {
	principal  auth.OperationPrincipal
	metadata   auth.RequestMetadata
	grantID    string
	breakGlass bool
}

type BreakGlassRequest struct {
	ReasonCode   string
	ReasonDetail *string
}

type BreakGlassGrant struct {
	GrantID   string    `json:"grantId"`
	ExpiresAt time.Time `json:"expiresAt"`
}

var ErrWarningOverflow = errors.New("patient summary warning item limit exceeded")

func NewService(pool *pgxpool.Pool, authorizer operationAuthorizer) (*Service, error) {
	if pool == nil || authorizer == nil {
		return nil, errors.New("patient summary database and authorizer are required")
	}
	return &Service{pool: pool, authorizer: authorizer, now: time.Now}, nil
}

func (s *Service) Authorize(ctx context.Context, token, csrf string, metadata auth.RequestMetadata) (Authorization, error) {
	principal, err := s.authorizer.AuthorizeOperation(ctx, auth.OperationAuthorizationRequest{
		Token: token, CSRFToken: csrf, Permission: permissionSummaryRead,
		DeniedEventType: "patient_summary_header.denied", Metadata: metadata,
	})
	if err != nil {
		return Authorization{}, err
	}
	return Authorization{principal: principal, metadata: metadata}, nil
}

func (s *Service) AuthorizeClinical(ctx context.Context, token, csrf string, metadata auth.RequestMetadata) (Authorization, error) {
	principal, err := s.authorizer.AuthorizeOperation(ctx, auth.OperationAuthorizationRequest{
		Token: token, CSRFToken: csrf, Permission: permissionClinicalSummaryRead,
		DeniedEventType: "patient_summary_warning.denied", Metadata: metadata,
	})
	if err != nil {
		if !errors.Is(err, auth.ErrForbidden) {
			return Authorization{}, err
		}
		principal, err = s.authorizer.AuthorizeOperation(ctx, auth.OperationAuthorizationRequest{
			Token: token, CSRFToken: csrf, Permission: permissionBreakGlass,
			DeniedEventType: "patient_break_glass.denied", Metadata: metadata,
		})
		if err != nil {
			return Authorization{}, err
		}
		return Authorization{principal: principal, metadata: metadata, breakGlass: true}, nil
	}
	return Authorization{principal: principal, metadata: metadata}, nil
}

func (s *Service) GetWarningDetails(ctx context.Context, authorization Authorization, publicID, contextVersion string) (WarningDetails, error) {
	if authorization.principal.UserID < 1 || contextVersion != strconv.FormatInt(authorization.principal.ContextVersion, 10) {
		return WarningDetails{}, auth.ErrConflict
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return WarningDetails{}, fmt.Errorf("begin warning transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	repository, err := NewRepository(tx)
	if err != nil {
		return WarningDetails{}, err
	}
	details, err := repository.LoadWarningDetails(ctx, authorization.principal.InstitutionID, publicID)
	if err != nil {
		return WarningDetails{}, err
	}
	grantID := authorization.grantID
	if grantID == "" {
		grantID, err = repository.LoadActiveGrantID(ctx, authorization.principal, publicID)
		if err != nil && !errors.Is(err, ErrGrantRequired) {
			return WarningDetails{}, err
		}
		if errors.Is(err, ErrGrantRequired) {
			grantID = ""
		}
	}
	if authorization.breakGlass && grantID == "" {
		return WarningDetails{}, ErrGrantRequired
	}
	attributes, _ := json.Marshal(map[string]any{"permission": permissionClinicalSummaryRead, "contextVersion": authorization.principal.ContextVersion})
	if grantID != "" {
		attributes, _ = json.Marshal(map[string]any{"permission": permissionBreakGlass, "grantId": grantID, "contextVersion": authorization.principal.ContextVersion})
	}
	if _, err := tx.Exec(ctx, `INSERT INTO audit_events (event_type, actor_user_id, session_id, institution_id, site_id, firm_id, outcome, reason_code, correlation_id, source_ip_class, attributes) VALUES ($1,$2,$3,$4,$5,$6,'success',$7,$8,$9,$10::jsonb)`, "patient_summary_warning.view", authorization.principal.UserID, authorization.principal.SessionID, authorization.principal.InstitutionID, authorization.principal.SiteID, authorization.principal.FirmID, "disclosed", authorization.metadata.CorrelationID, authorization.metadata.SourceIPClass, attributes); err != nil {
		return WarningDetails{}, fmt.Errorf("audit patient warnings: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return WarningDetails{}, fmt.Errorf("commit patient warnings: %w", err)
	}
	return details, nil
}

func (s *Service) CreateBreakGlassGrant(ctx context.Context, token, csrf string, metadata auth.RequestMetadata, publicID, contextVersion string, request BreakGlassRequest) (BreakGlassGrant, error) {
	if !validBreakGlassReason(request.ReasonCode) || (request.ReasonCode == "other" && (request.ReasonDetail == nil || *request.ReasonDetail == "")) {
		return BreakGlassGrant{}, ErrInvalidRequest
	}
	authorization, err := s.authorizePermission(ctx, token, csrf, permissionBreakGlass, "patient_break_glass.denied", metadata)
	if err != nil {
		return BreakGlassGrant{}, err
	}
	if contextVersion != strconv.FormatInt(authorization.principal.ContextVersion, 10) {
		return BreakGlassGrant{}, auth.ErrConflict
	}
	grantID, err := newGrantID()
	if err != nil {
		return BreakGlassGrant{}, fmt.Errorf("create break-glass grant id: %w", err)
	}
	now := s.now().UTC()
	expires := now.Add(15 * time.Minute)
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return BreakGlassGrant{}, fmt.Errorf("begin break-glass transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var patientID int64
	if err := tx.QueryRow(ctx, `SELECT p.id FROM patients p JOIN patient_institutions pi ON pi.patient_id = p.id AND pi.institution_id = $1 AND pi.active WHERE p.public_id = $2 AND p.active`, authorization.principal.InstitutionID, publicID).Scan(&patientID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return BreakGlassGrant{}, ErrNotFound
		}
		return BreakGlassGrant{}, fmt.Errorf("resolve break-glass patient: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO patient_break_glass_grants (grant_id, patient_id, user_id, session_id, institution_id, site_id, firm_id, reason_code, reason_detail, created_at, expires_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, grantID, patientID, authorization.principal.UserID, authorization.principal.SessionID, authorization.principal.InstitutionID, authorization.principal.SiteID, authorization.principal.FirmID, request.ReasonCode, request.ReasonDetail, now, expires); err != nil {
		return BreakGlassGrant{}, fmt.Errorf("store break-glass grant: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO patient_break_glass_alert_outbox (grant_id, event_type) VALUES ($1, 'created')`, grantID); err != nil {
		return BreakGlassGrant{}, fmt.Errorf("queue break-glass alert: %w", err)
	}
	attributes, _ := json.Marshal(map[string]any{"grantId": grantID, "reasonCode": request.ReasonCode, "expiresAt": expires})
	if _, err := tx.Exec(ctx, `INSERT INTO audit_events (event_type, actor_user_id, session_id, institution_id, site_id, firm_id, outcome, reason_code, correlation_id, source_ip_class, attributes) VALUES ($1,$2,$3,$4,$5,$6,'success',$7,$8,$9,$10::jsonb)`, "patient.break_glass.granted", authorization.principal.UserID, authorization.principal.SessionID, authorization.principal.InstitutionID, authorization.principal.SiteID, authorization.principal.FirmID, request.ReasonCode, metadata.CorrelationID, metadata.SourceIPClass, attributes); err != nil {
		return BreakGlassGrant{}, fmt.Errorf("audit break-glass grant: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return BreakGlassGrant{}, fmt.Errorf("commit break-glass grant: %w", err)
	}
	return BreakGlassGrant{GrantID: grantID, ExpiresAt: expires}, nil
}

func validBreakGlassReason(value string) bool {
	switch value {
	case "community_optometrist", "emergency_care", "on_behalf_of_another_area", "administrative_support", "other":
		return true
	default:
		return false
	}
}

func (s *Service) RevokeBreakGlassGrant(ctx context.Context, token, csrf string, metadata auth.RequestMetadata, patientID, grantID, contextVersion string) error {
	authorization, err := s.authorizePermission(ctx, token, csrf, permissionBreakGlassRevoke, "patient_break_glass.revoke_denied", metadata)
	if err != nil {
		return err
	}
	if contextVersion != strconv.FormatInt(authorization.principal.ContextVersion, 10) {
		return auth.ErrConflict
	}
	now := s.now().UTC()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin break-glass revocation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	result, err := tx.Exec(ctx, `UPDATE patient_break_glass_grants g SET revoked_at = $1 FROM patients p WHERE g.grant_id = $2 AND g.patient_id = p.id AND p.public_id = $3 AND g.institution_id = $4 AND g.revoked_at IS NULL`, now, grantID, patientID, authorization.principal.InstitutionID)
	if err != nil {
		return fmt.Errorf("revoke break-glass grant: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrNotFound
	}
	if _, err := tx.Exec(ctx, `INSERT INTO patient_break_glass_alert_outbox (grant_id, event_type) VALUES ($1, 'revoked')`, grantID); err != nil {
		return fmt.Errorf("queue break-glass revocation alert: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO audit_events (event_type, actor_user_id, session_id, institution_id, site_id, firm_id, outcome, reason_code, correlation_id, source_ip_class, attributes) VALUES ($1,$2,$3,$4,$5,$6,'success','revoked',$7,$8,'{}'::jsonb)`, "patient.break_glass.revoked", authorization.principal.UserID, authorization.principal.SessionID, authorization.principal.InstitutionID, authorization.principal.SiteID, authorization.principal.FirmID, metadata.CorrelationID, metadata.SourceIPClass); err != nil {
		return fmt.Errorf("audit break-glass revocation: %w", err)
	}
	return tx.Commit(ctx)
}

func (s *Service) authorizePermission(ctx context.Context, token, csrf, permission, deniedEvent string, metadata auth.RequestMetadata) (Authorization, error) {
	principal, err := s.authorizer.AuthorizeOperation(ctx, auth.OperationAuthorizationRequest{Token: token, CSRFToken: csrf, Permission: permission, DeniedEventType: deniedEvent, Metadata: metadata})
	if err != nil {
		return Authorization{}, err
	}
	return Authorization{principal: principal, metadata: metadata}, nil
}

func newGrantID() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func (s *Service) GetHeader(ctx context.Context, authorization Authorization, publicID, contextVersion string) (Header, error) {
	if authorization.principal.UserID < 1 || publicID == "" {
		return Header{}, auth.ErrForbidden
	}
	if contextVersion != strconv.FormatInt(authorization.principal.ContextVersion, 10) {
		return Header{}, auth.ErrConflict
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return Header{}, fmt.Errorf("begin patient summary transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	repository, err := NewRepository(tx)
	if err != nil {
		return Header{}, err
	}
	header, err := repository.LoadHeader(ctx, authorization.principal.InstitutionID, publicID)
	if err != nil {
		return Header{}, err
	}
	comparisonDate := s.now().UTC()
	if header.Deceased && header.DateOfDeath != nil {
		comparisonDate = *header.DateOfDeath
	}
	age := comparisonDate.Year() - header.DateOfBirth.Year()
	if comparisonDate.YearDay() < header.DateOfBirth.YearDay() {
		age--
	}
	if age >= 0 {
		header.AgeYears = &age
	}
	attributes, _ := json.Marshal(map[string]any{
		"permission":     permissionSummaryRead,
		"contextVersion": authorization.principal.ContextVersion,
	})
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (
			event_type, actor_user_id, session_id, institution_id, site_id, firm_id,
			outcome, reason_code, correlation_id, source_ip_class, attributes
		) VALUES ($1, $2, $3, $4, $5, $6, 'success', $7, $8, $9, $10::jsonb)`,
		"patient_summary_header.view", authorization.principal.UserID,
		authorization.principal.SessionID, authorization.principal.InstitutionID,
		authorization.principal.SiteID, authorization.principal.FirmID,
		"disclosed", authorization.metadata.CorrelationID,
		authorization.metadata.SourceIPClass, attributes,
	); err != nil {
		return Header{}, fmt.Errorf("audit patient summary header: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Header{}, fmt.Errorf("commit patient summary header: %w", err)
	}
	return header, nil
}
