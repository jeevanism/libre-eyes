package theatrebooking

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jeevanism/visionopus/internal/auth"
)

type operationAuthorizer interface {
	AuthorizeOperation(context.Context, auth.OperationAuthorizationRequest) (auth.OperationPrincipal, error)
	AuthorizeRead(context.Context, auth.ReadAuthorizationRequest) (auth.OperationPrincipal, error)
}

// Authorization proves this service derived the actor and context from a session.
type Authorization struct {
	principal auth.OperationPrincipal
	metadata  auth.RequestMetadata
	service   *Service
}

type Service struct {
	pool       *pgxpool.Pool
	authorizer operationAuthorizer
}

const developmentFixtureSessionDate = "2026-08-10"

func NewService(pool *pgxpool.Pool, authorizer operationAuthorizer) (*Service, error) {
	if pool == nil || authorizer == nil {
		return nil, errors.New("theatre booking database and authorizer are required")
	}
	return &Service{pool: pool, authorizer: authorizer}, nil
}

func (s *Service) AuthorizeRead(ctx context.Context, token string, metadata auth.RequestMetadata) (Authorization, error) {
	principal, err := s.authorizer.AuthorizeRead(ctx, auth.ReadAuthorizationRequest{
		Token: token, Permission: PermissionManage, DeniedEventType: "theatre.development_booking.denied", Metadata: metadata,
	})
	if err != nil {
		return Authorization{}, err
	}
	return Authorization{principal: principal, metadata: metadata, service: s}, nil
}

func (s *Service) AuthorizeCommand(ctx context.Context, token, csrf string, metadata auth.RequestMetadata) (Authorization, error) {
	principal, err := s.authorizer.AuthorizeOperation(ctx, auth.OperationAuthorizationRequest{
		Token: token, CSRFToken: csrf, Permission: PermissionManage, DeniedEventType: "theatre.development_booking.denied", Metadata: metadata,
	})
	if err != nil {
		return Authorization{}, err
	}
	return Authorization{principal: principal, metadata: metadata, service: s}, nil
}

func (s *Service) Board(ctx context.Context, authorization Authorization, dateFilter, roomID string) (Board, error) {
	if !s.validAuthorization(authorization) || !validOptionalDate(dateFilter) || roomID != "" && !validUUIDv4(roomID) {
		return Board{}, ErrInvalidRequest
	}
	args := []any{authorization.principal.InstitutionID, authorization.principal.SiteID, authorization.principal.FirmID}
	where := "s.institution_id = $1 AND s.site_id = $2 AND s.firm_id = $3"
	if dateFilter != "" {
		args = append(args, dateFilter)
		where += fmt.Sprintf(" AND s.starts_at::date = $%d::date", len(args))
	}
	if roomID != "" {
		args = append(args, roomID)
		where += fmt.Sprintf(" AND s.room_id = $%d::uuid", len(args))
	}
	query := fmt.Sprintf(`
		SELECT s.id::text, r.synthetic_label, s.starts_at, s.ends_at, s.capacity_minutes,
			COALESCE(SUM(CASE WHEN b.status = 'scheduled' THEN b.requested_duration_minutes ELSE 0 END), 0)::int, s.version
		FROM development_theatre_sessions s
		JOIN development_theatre_rooms r ON r.id = s.room_id
		LEFT JOIN development_booking_requests b ON b.assigned_session_id = s.id AND b.status = 'scheduled'
		WHERE %s
		GROUP BY s.id, r.synthetic_label
		ORDER BY s.starts_at, s.id`, where)
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return Board{}, fmt.Errorf("list development theatre sessions: %w", err)
	}
	defer rows.Close()
	board := Board{DevelopmentOnly: true, TheatreSessions: []Session{}, BookingRequests: []BookingRequest{}, Whiteboard: []WhiteboardGroup{}}
	allowedSessions := make(map[string]Session)
	for rows.Next() {
		var session Session
		var startsAt, endsAt time.Time
		if err := rows.Scan(&session.ID, &session.SyntheticRoomLabel, &startsAt, &endsAt, &session.CapacityMinutes, &session.AllocatedMinutes, &session.Version); err != nil {
			return Board{}, fmt.Errorf("scan development theatre session: %w", err)
		}
		session.StartsAt, session.EndsAt = startsAt.UTC().Format(time.RFC3339), endsAt.UTC().Format(time.RFC3339)
		if board.SessionDate == "" {
			board.SessionDate = startsAt.UTC().Format("2006-01-02")
		}
		board.TheatreSessions = append(board.TheatreSessions, session)
		allowedSessions[session.ID] = session
	}
	if err := rows.Err(); err != nil {
		return Board{}, fmt.Errorf("iterate development theatre sessions: %w", err)
	}
	if board.SessionDate == "" {
		if dateFilter != "" {
			board.SessionDate = dateFilter
		} else {
			board.SessionDate = developmentFixtureSessionDate
		}
	}
	requestRows, err := s.pool.Query(ctx, `
		SELECT id::text, synthetic_label, requested_duration_minutes, status, assigned_session_id::text, version
		FROM development_booking_requests
		WHERE institution_id = $1 AND site_id = $2 AND firm_id = $3
		ORDER BY created_at, id`, authorization.principal.InstitutionID, authorization.principal.SiteID, authorization.principal.FirmID)
	if err != nil {
		return Board{}, fmt.Errorf("list development booking requests: %w", err)
	}
	defer requestRows.Close()
	whiteboard := make(map[string][]WhiteboardEntry)
	for requestRows.Next() {
		var item BookingRequest
		if err := requestRows.Scan(&item.ID, &item.SyntheticLabel, &item.RequestedDurationMinutes, &item.Status, &item.AssignedSessionID, &item.Version); err != nil {
			return Board{}, fmt.Errorf("scan development booking request: %w", err)
		}
		if item.AssignedSessionID != nil {
			if _, visible := allowedSessions[*item.AssignedSessionID]; !visible && (dateFilter != "" || roomID != "") {
				// A filtered board must not return a scheduled request whose session
				// is absent from the same response.
				continue
			}
		}
		board.BookingRequests = append(board.BookingRequests, item)
		if item.Status == StatusScheduled && item.AssignedSessionID != nil {
			whiteboard[*item.AssignedSessionID] = append(whiteboard[*item.AssignedSessionID], WhiteboardEntry{RequestID: item.ID, SyntheticLabel: item.SyntheticLabel, RequestedDurationMinutes: item.RequestedDurationMinutes})
		}
	}
	if err := requestRows.Err(); err != nil {
		return Board{}, fmt.Errorf("iterate development booking requests: %w", err)
	}
	for _, session := range board.TheatreSessions {
		board.Whiteboard = append(board.Whiteboard, WhiteboardGroup{SessionID: session.ID, SyntheticRoomLabel: session.SyntheticRoomLabel, Entries: whiteboard[session.ID]})
	}
	return board, nil
}

func (s *Service) Command(ctx context.Context, authorization Authorization, request CommandRequest) (BookingRequest, error) {
	if !s.validAuthorization(authorization) || !validUUIDv4(request.RequestID) || request.ExpectedVersion < 1 || !validCommand(request.Command) || (request.Command != CommandCancel && !validUUIDv4(request.TargetSessionID)) {
		return BookingRequest{}, ErrInvalidRequest
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return BookingRequest{}, fmt.Errorf("begin development theatre command: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Read without a lock only to determine every relevant session. The session rows
	// are then locked in lexical order before the booking request and revalidated.
	observed, err := loadRequest(ctx, tx, authorization.principal, request.RequestID, false)
	if errors.Is(err, pgx.ErrNoRows) {
		return BookingRequest{}, ErrNotFound
	}
	if err != nil {
		return BookingRequest{}, fmt.Errorf("load development booking request: %w", err)
	}
	current := observed
	ids := involvedSessionIDs(observed.AssignedSessionID, request.TargetSessionID, request.Command)
	locked, err := lockSessions(ctx, tx, authorization.principal, ids)
	if errors.Is(err, pgx.ErrNoRows) {
		return BookingRequest{}, ErrNotFound
	}
	if err != nil {
		return BookingRequest{}, fmt.Errorf("lock development theatre sessions: %w", err)
	}
	current, err = loadRequest(ctx, tx, authorization.principal, request.RequestID, true)
	if errors.Is(err, pgx.ErrNoRows) {
		return BookingRequest{}, ErrNotFound
	}
	if err != nil {
		return BookingRequest{}, fmt.Errorf("lock development booking request: %w", err)
	}
	if !sameBookingState(current, observed) {
		return BookingRequest{}, ConflictError{Reason: "stale_version"}
	}
	if current.Version != request.ExpectedVersion {
		return BookingRequest{}, ConflictError{Reason: "stale_version"}
	}
	if err := validateTransition(current, request); err != nil {
		return BookingRequest{}, err
	}
	if request.Command != CommandCancel {
		target := locked[request.TargetSessionID]
		allocated, err := allocatedMinutes(ctx, tx, request.TargetSessionID, current.ID)
		if err != nil {
			return BookingRequest{}, fmt.Errorf("calculate development theatre capacity: %w", err)
		}
		if allocated+current.RequestedDurationMinutes > target.CapacityMinutes {
			return BookingRequest{}, ConflictError{Reason: "insufficient_capacity"}
		}
	}
	priorStatus, priorSessionID := current.Status, current.AssignedSessionID
	nextStatus, nextSessionID := nextAssignment(request)
	updated, err := updateRequest(ctx, tx, authorization.principal, current, nextStatus, nextSessionID)
	if err != nil {
		return BookingRequest{}, err
	}
	if err := bumpSessions(ctx, tx, ids); err != nil {
		return BookingRequest{}, err
	}
	if err := appendAudit(ctx, tx, authorization, request.Command, priorStatus, nextStatus, priorSessionID, nextSessionID, updated); err != nil {
		return BookingRequest{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return BookingRequest{}, fmt.Errorf("commit development theatre command: %w", err)
	}
	return updated, nil
}

func (s *Service) validAuthorization(value Authorization) bool {
	return value.service == s && value.principal.UserID > 0
}

func loadRequest(ctx context.Context, tx pgx.Tx, principal auth.OperationPrincipal, id string, forUpdate bool) (BookingRequest, error) {
	lock := ""
	if forUpdate {
		lock = " FOR UPDATE"
	}
	var item BookingRequest
	err := tx.QueryRow(ctx, `SELECT id::text, synthetic_label, requested_duration_minutes, status, assigned_session_id::text, version FROM development_booking_requests WHERE id = $1::uuid AND institution_id = $2 AND site_id = $3 AND firm_id = $4`+lock, id, principal.InstitutionID, principal.SiteID, principal.FirmID).Scan(&item.ID, &item.SyntheticLabel, &item.RequestedDurationMinutes, &item.Status, &item.AssignedSessionID, &item.Version)
	return item, err
}

func lockSessions(ctx context.Context, tx pgx.Tx, principal auth.OperationPrincipal, ids []string) (map[string]Session, error) {
	locked := make(map[string]Session, len(ids))
	for _, id := range ids {
		var session Session
		err := tx.QueryRow(ctx, `SELECT s.id::text, r.synthetic_label, s.starts_at, s.ends_at, s.capacity_minutes, 0, s.version FROM development_theatre_sessions s JOIN development_theatre_rooms r ON r.id = s.room_id WHERE s.id = $1::uuid AND s.institution_id = $2 AND s.site_id = $3 AND s.firm_id = $4 FOR UPDATE OF s`, id, principal.InstitutionID, principal.SiteID, principal.FirmID).Scan(&session.ID, &session.SyntheticRoomLabel, new(time.Time), new(time.Time), &session.CapacityMinutes, &session.AllocatedMinutes, &session.Version)
		if err != nil {
			return nil, err
		}
		locked[id] = session
	}
	return locked, nil
}

func allocatedMinutes(ctx context.Context, tx pgx.Tx, sessionID, excludedRequestID string) (int, error) {
	var minutes int
	err := tx.QueryRow(ctx, `SELECT COALESCE(SUM(requested_duration_minutes), 0)::int FROM development_booking_requests WHERE assigned_session_id = $1::uuid AND status = 'scheduled' AND id <> $2::uuid`, sessionID, excludedRequestID).Scan(&minutes)
	return minutes, err
}

func updateRequest(ctx context.Context, tx pgx.Tx, principal auth.OperationPrincipal, current BookingRequest, status Status, sessionID *string) (BookingRequest, error) {
	var updated BookingRequest
	err := tx.QueryRow(ctx, `UPDATE development_booking_requests SET status = $1::development_booking_request_status, assigned_session_id = $2::uuid, version = version + 1, updated_at = now() WHERE id = $3::uuid AND institution_id = $4 AND site_id = $5 AND firm_id = $6 AND version = $7 RETURNING id::text, synthetic_label, requested_duration_minutes, status, assigned_session_id::text, version`, status, sessionID, current.ID, principal.InstitutionID, principal.SiteID, principal.FirmID, current.Version).Scan(&updated.ID, &updated.SyntheticLabel, &updated.RequestedDurationMinutes, &updated.Status, &updated.AssignedSessionID, &updated.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return BookingRequest{}, ConflictError{Reason: "stale_version"}
	}
	if err != nil {
		return BookingRequest{}, fmt.Errorf("update development booking request: %w", err)
	}
	return updated, nil
}

func bumpSessions(ctx context.Context, tx pgx.Tx, ids []string) error {
	for _, id := range ids {
		if _, err := tx.Exec(ctx, `UPDATE development_theatre_sessions SET version = version + 1, updated_at = now() WHERE id = $1::uuid`, id); err != nil {
			return fmt.Errorf("update development theatre session: %w", err)
		}
	}
	return nil
}

func appendAudit(ctx context.Context, tx pgx.Tx, authorization Authorization, command Command, prior, next Status, priorSessionID, nextSessionID *string, item BookingRequest) error {
	_, err := tx.Exec(ctx, `INSERT INTO development_theatre_booking_audit (request_id, actor_user_id, institution_id, site_id, firm_id, command, prior_state, next_state, prior_session_id, next_session_id, resulting_version, correlation_id) VALUES ($1::uuid,$2,$3,$4,$5,$6,$7::development_booking_request_status,$8::development_booking_request_status,$9::uuid,$10::uuid,$11,$12)`, item.ID, authorization.principal.UserID, authorization.principal.InstitutionID, authorization.principal.SiteID, authorization.principal.FirmID, command, prior, next, priorSessionID, nextSessionID, item.Version, authorization.metadata.CorrelationID)
	if err != nil {
		return fmt.Errorf("append development theatre audit: %w", err)
	}
	return nil
}

func validateTransition(current BookingRequest, request CommandRequest) error {
	switch request.Command {
	case CommandSchedule:
		if current.Status != StatusWaiting || current.AssignedSessionID != nil {
			return ConflictError{Reason: "invalid_state"}
		}
	case CommandReschedule:
		if current.Status != StatusScheduled || current.AssignedSessionID == nil || *current.AssignedSessionID == request.TargetSessionID {
			return ConflictError{Reason: "invalid_state"}
		}
	case CommandCancel:
		if current.Status != StatusWaiting && current.Status != StatusScheduled {
			return ConflictError{Reason: "invalid_state"}
		}
	default:
		return ErrInvalidRequest
	}
	return nil
}

func nextAssignment(request CommandRequest) (Status, *string) {
	if request.Command == CommandCancel {
		return StatusCancelled, nil
	}
	return StatusScheduled, &request.TargetSessionID
}

func involvedSessionIDs(prior *string, target string, command Command) []string {
	ids := make(map[string]bool)
	if prior != nil {
		ids[*prior] = true
	}
	if command != CommandCancel {
		ids[target] = true
	}
	values := make([]string, 0, len(ids))
	for id := range ids {
		values = append(values, id)
	}
	sort.Strings(values)
	return values
}

func sameBookingState(current, observed BookingRequest) bool {
	return current.Version == observed.Version &&
		current.Status == observed.Status &&
		sameOptionalString(current.AssignedSessionID, observed.AssignedSessionID)
}

func sameOptionalString(left, right *string) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}

func validOptionalDate(value string) bool {
	if value == "" {
		return true
	}
	_, err := time.Parse("2006-01-02", value)
	return err == nil
}
func validCommand(value Command) bool {
	return value == CommandSchedule || value == CommandReschedule || value == CommandCancel
}
func validUUIDv4(value string) bool {
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
