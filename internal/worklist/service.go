package worklist

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jeevanism/visionopus/internal/auth"
)

type operationAuthorizer interface {
	AuthorizeOperation(context.Context, auth.OperationAuthorizationRequest) (auth.OperationPrincipal, error)
}

// Authorization proves this service authorized the development queue operation.
type Authorization struct {
	principal auth.OperationPrincipal
	metadata  auth.RequestMetadata
	service   *Service
}

// Service owns the synthetic development clinic-flow boundary.
type Service struct {
	pool       *pgxpool.Pool
	authorizer operationAuthorizer
}

func NewService(pool *pgxpool.Pool, authorizer operationAuthorizer) (*Service, error) {
	if pool == nil || authorizer == nil {
		return nil, errors.New("worklist database and authorizer are required")
	}
	return &Service{pool: pool, authorizer: authorizer}, nil
}

func (s *Service) Authorize(ctx context.Context, token, csrf string, metadata auth.RequestMetadata) (Authorization, error) {
	principal, err := s.authorizer.AuthorizeOperation(ctx, auth.OperationAuthorizationRequest{
		Token: token, CSRFToken: csrf, Permission: PermissionManage,
		Capability:      "clinic_flow",
		DeniedEventType: "worklist.development_flow.denied", Metadata: metadata,
	})
	if err != nil {
		return Authorization{}, err
	}
	return Authorization{principal: principal, metadata: metadata, service: s}, nil
}

func (s *Service) List(ctx context.Context, authorization Authorization) ([]Ticket, error) {
	if !s.validAuthorization(authorization) {
		return nil, ErrInvalidRequest
	}
	rows, err := s.pool.Query(ctx, `
		SELECT t.id::text, t.synthetic_patient_label, t.status, u.display_name, t.version
		FROM development_flow_tickets t
		LEFT JOIN users u ON u.id = t.assignee_user_id
		WHERE t.institution_id = $1 AND t.site_id = $2 AND t.firm_id = $3
		ORDER BY t.created_at, t.id`, authorization.principal.InstitutionID, authorization.principal.SiteID, authorization.principal.FirmID)
	if err != nil {
		return nil, fmt.Errorf("list development flow tickets: %w", err)
	}
	defer rows.Close()

	tickets := make([]Ticket, 0)
	for rows.Next() {
		var ticket Ticket
		if err := rows.Scan(&ticket.ID, &ticket.SyntheticPatientLabel, &ticket.Status, &ticket.AssigneeDisplayName, &ticket.Version); err != nil {
			return nil, fmt.Errorf("scan development flow ticket: %w", err)
		}
		tickets = append(tickets, ticket)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate development flow tickets: %w", err)
	}
	return tickets, nil
}

func (s *Service) Command(ctx context.Context, authorization Authorization, request CommandRequest) (Ticket, error) {
	if !s.validAuthorization(authorization) || !validTicketID(request.TicketID) || request.ExpectedVersion < 1 || !validCommand(request.Command) {
		return Ticket{}, ErrInvalidRequest
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return Ticket{}, fmt.Errorf("begin development flow command: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	current, assigneeID, err := loadForUpdate(ctx, tx, authorization.principal, request.TicketID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Ticket{}, ErrNotFound
	}
	if err != nil {
		return Ticket{}, fmt.Errorf("load development flow ticket: %w", err)
	}
	if current.Version != request.ExpectedVersion {
		return Ticket{}, ErrConflict
	}
	next, nextAssigneeID, err := transition(current.Status, assigneeID, authorization.principal.UserID, request.Command)
	if err != nil {
		return Ticket{}, err
	}
	updated, err := updateTicket(ctx, tx, authorization.principal, request, assigneeID, next, nextAssigneeID)
	if err != nil {
		return Ticket{}, err
	}
	if err := appendAudit(ctx, tx, authorization, request.Command, current.Status, next, updated.Version, updated.ID); err != nil {
		return Ticket{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Ticket{}, fmt.Errorf("commit development flow command: %w", err)
	}
	return updated, nil
}

func loadForUpdate(ctx context.Context, tx pgx.Tx, principal auth.OperationPrincipal, ticketID string) (Ticket, *int64, error) {
	var ticket Ticket
	var assigneeID *int64
	err := tx.QueryRow(ctx, `
		SELECT t.id::text, t.synthetic_patient_label, t.status, u.display_name, t.version, t.assignee_user_id
		FROM development_flow_tickets t
		LEFT JOIN users u ON u.id = t.assignee_user_id
		WHERE t.id = $1::uuid AND t.institution_id = $2 AND t.site_id = $3 AND t.firm_id = $4
		FOR UPDATE OF t`, ticketID, principal.InstitutionID, principal.SiteID, principal.FirmID,
	).Scan(&ticket.ID, &ticket.SyntheticPatientLabel, &ticket.Status, &ticket.AssigneeDisplayName, &ticket.Version, &assigneeID)
	return ticket, assigneeID, err
}

func transition(current Status, assigneeID *int64, actorID int64, command Command) (Status, *int64, error) {
	switch command {
	case CommandArrive:
		if current != StatusWaiting || assigneeID != nil {
			return "", nil, ErrInvalidTransition
		}
		return StatusArrived, nil, nil
	case CommandClaim:
		if current != StatusArrived || assigneeID != nil {
			return "", nil, ErrInvalidTransition
		}
		return StatusInProgress, &actorID, nil
	case CommandRelease:
		if current != StatusInProgress {
			return "", nil, ErrInvalidTransition
		}
		if assigneeID == nil || *assigneeID != actorID {
			return "", nil, ErrNotFound
		}
		return StatusArrived, nil, nil
	case CommandComplete:
		if current != StatusInProgress {
			return "", nil, ErrInvalidTransition
		}
		if assigneeID == nil || *assigneeID != actorID {
			return "", nil, ErrNotFound
		}
		return StatusCompleted, assigneeID, nil
	default:
		return "", nil, ErrInvalidRequest
	}
}

func updateTicket(ctx context.Context, tx pgx.Tx, principal auth.OperationPrincipal, request CommandRequest, currentAssigneeID *int64, next Status, nextAssigneeID *int64) (Ticket, error) {
	var ticket Ticket
	err := tx.QueryRow(ctx, `
		WITH updated AS (
			UPDATE development_flow_tickets t
			SET status = $1::development_flow_ticket_status, assignee_user_id = $2, version = version + 1, updated_at = now()
			WHERE t.id = $3::uuid AND t.institution_id = $4 AND t.site_id = $5 AND t.firm_id = $6
				AND t.version = $7 AND (t.assignee_user_id IS NOT DISTINCT FROM $8)
			RETURNING t.id, t.synthetic_patient_label, t.status, t.assignee_user_id, t.version
		)
		SELECT updated.id::text, updated.synthetic_patient_label, updated.status, u.display_name, updated.version
		FROM updated LEFT JOIN users u ON u.id = updated.assignee_user_id`,
		next, nextAssigneeID, request.TicketID, principal.InstitutionID, principal.SiteID, principal.FirmID, request.ExpectedVersion, currentAssigneeID,
	).Scan(&ticket.ID, &ticket.SyntheticPatientLabel, &ticket.Status, &ticket.AssigneeDisplayName, &ticket.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return Ticket{}, ErrConflict
	}
	if err != nil {
		return Ticket{}, fmt.Errorf("update development flow ticket: %w", err)
	}
	return ticket, nil
}

func appendAudit(ctx context.Context, tx pgx.Tx, authorization Authorization, command Command, prior, next Status, version int64, ticketID string) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO development_flow_audit (
			ticket_id, actor_user_id, institution_id, site_id, firm_id, command,
			prior_state, next_state, resulting_version, correlation_id
		) VALUES ($1::uuid,$2,$3,$4,$5,$6,$7::development_flow_ticket_status,$8::development_flow_ticket_status,$9,$10)`,
		ticketID, authorization.principal.UserID, authorization.principal.InstitutionID,
		authorization.principal.SiteID, authorization.principal.FirmID, command, prior, next,
		version, authorization.metadata.CorrelationID,
	); err != nil {
		return fmt.Errorf("append development flow audit: %w", err)
	}
	return nil
}

func (s *Service) validAuthorization(authorization Authorization) bool {
	return authorization.service == s && authorization.principal.UserID > 0
}

func validTicketID(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' || value[14] != '4' || !strings.ContainsRune("89ab", rune(value[19])) {
		return false
	}
	for index, character := range value {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			continue
		}
		if !strings.ContainsRune("0123456789abcdef", character) {
			return false
		}
	}
	return true
}

func validCommand(command Command) bool {
	return command == CommandArrive || command == CommandClaim || command == CommandRelease || command == CommandComplete
}
