package patientsummary

import (
	"context"
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

type operationAuthorizer interface {
	AuthorizeOperation(context.Context, auth.OperationAuthorizationRequest) (auth.OperationPrincipal, error)
}

type Service struct {
	pool       *pgxpool.Pool
	authorizer operationAuthorizer
	now        func() time.Time
}

type Authorization struct {
	principal auth.OperationPrincipal
	metadata  auth.RequestMetadata
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
		return Authorization{}, err
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
	attributes, _ := json.Marshal(map[string]any{"permission": permissionClinicalSummaryRead, "contextVersion": authorization.principal.ContextVersion})
	if _, err := tx.Exec(ctx, `INSERT INTO audit_events (event_type, actor_user_id, session_id, institution_id, site_id, firm_id, outcome, reason_code, correlation_id, source_ip_class, attributes) VALUES ($1,$2,$3,$4,$5,$6,'success',$7,$8,$9,$10::jsonb)`, "patient_summary_warning.view", authorization.principal.UserID, authorization.principal.SessionID, authorization.principal.InstitutionID, authorization.principal.SiteID, authorization.principal.FirmID, "disclosed", authorization.metadata.CorrelationID, authorization.metadata.SourceIPClass, attributes); err != nil {
		return WarningDetails{}, fmt.Errorf("audit patient warnings: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return WarningDetails{}, fmt.Errorf("commit patient warnings: %w", err)
	}
	return details, nil
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
