package auth

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	permissionReadSession   = "session.read_self"
	permissionRevokeSession = "session.revoke_self"
	permissionSwitchContext = "context.switch"
)

// ServiceConfig controls authentication policy established by ADR-0003.
type ServiceConfig struct {
	CSRFKey                []byte
	SessionIdleTimeout     time.Duration
	SessionAbsoluteTimeout time.Duration
	LoginFailureLimit      int
	SoftLockDuration       time.Duration
}

// Service implements authentication use cases against PostgreSQL.
type Service struct {
	pool      *pgxpool.Pool
	passwords PasswordManager
	config    ServiceConfig
	now       func() time.Time
	dummyHash string
}

// NewService constructs the authentication service and its timing-safe dummy hash.
func NewService(pool *pgxpool.Pool, cfg ServiceConfig) (*Service, error) {
	if pool == nil {
		return nil, errors.New("authentication database is required")
	}
	if len(cfg.CSRFKey) < 32 {
		return nil, errors.New("authentication CSRF key must contain at least 32 bytes")
	}
	if cfg.SessionIdleTimeout <= 0 || cfg.SessionAbsoluteTimeout <= 0 || cfg.SoftLockDuration <= 0 {
		return nil, errors.New("authentication timeouts must be positive")
	}
	if cfg.SessionIdleTimeout > cfg.SessionAbsoluteTimeout {
		return nil, errors.New("session idle timeout must not exceed absolute timeout")
	}
	if cfg.LoginFailureLimit < 1 {
		return nil, errors.New("login failure limit must be positive")
	}

	passwords := PasswordManager{}
	dummyHash, err := passwords.Hash("visionopus timing equalization value")
	if err != nil {
		return nil, fmt.Errorf("create authentication dummy hash: %w", err)
	}
	return &Service{
		pool:      pool,
		passwords: passwords,
		config:    cfg,
		now:       time.Now,
		dummyHash: dummyHash,
	}, nil
}

// ListLoginOptions returns active institutions and sites without user data.
func (s *Service) ListLoginOptions(ctx context.Context) ([]InstitutionOption, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT i.id, i.name, st.id, st.name
		FROM institutions i
		JOIN sites st ON st.institution_id = i.id AND st.active
		WHERE i.active
		ORDER BY lower(i.name), i.id, lower(st.name), st.id`)
	if err != nil {
		return nil, fmt.Errorf("list login options: %w", err)
	}
	defer rows.Close()

	options := make([]InstitutionOption, 0)
	byInstitution := make(map[int64]int)
	for rows.Next() {
		var institution, site Reference
		if err := rows.Scan(&institution.ID, &institution.Name, &site.ID, &site.Name); err != nil {
			return nil, fmt.Errorf("scan login option: %w", err)
		}
		index, ok := byInstitution[institution.ID]
		if !ok {
			index = len(options)
			byInstitution[institution.ID] = index
			options = append(options, InstitutionOption{Institution: institution, Sites: make([]Reference, 0, 1)})
		}
		options[index].Sites = append(options[index].Sites, site)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate login options: %w", err)
	}
	return options, nil
}

// Login authenticates a local credential and atomically creates a complete session.
func (s *Service) Login(ctx context.Context, request LoginRequest, metadata RequestMetadata) (CreatedSession, error) {
	now := s.now().UTC()
	username := strings.ToLower(strings.TrimSpace(request.Username))
	token, err := newToken()
	if err != nil {
		return CreatedSession{}, err
	}
	tokenHash := tokenDigest(token)
	csrf := csrfToken(s.config.CSRFKey, token)
	csrfHash := tokenDigest(csrf)

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return CreatedSession{}, fmt.Errorf("begin login transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	credential, err := loadCredentialForLogin(ctx, tx, username, request.InstitutionID)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return CreatedSession{}, fmt.Errorf("load login credential: %w", err)
		}
		_, _ = s.passwords.Verify(s.dummyHash, request.Password)
		if err := insertAudit(ctx, tx, auditRecord{
			eventType: "auth.login_failed", outcome: "failure", reasonCode: "credential_not_found", metadata: metadata,
		}); err != nil {
			return CreatedSession{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return CreatedSession{}, fmt.Errorf("commit failed login audit: %w", err)
		}
		return CreatedSession{}, ErrAuthenticationFailed
	}

	if credential.state == "soft_locked" && credential.softLockedUntil != nil && !credential.softLockedUntil.After(now) {
		if _, err := tx.Exec(ctx, `
			UPDATE user_credentials
			SET state = 'active', failed_attempts = 0, soft_locked_until = NULL,
				version = version + 1, updated_at = $2
			WHERE id = $1`, credential.credentialID, now); err != nil {
			return CreatedSession{}, fmt.Errorf("release expired soft lock: %w", err)
		}
		credential.state = "active"
		credential.failedAttempts = 0
		credential.softLockedUntil = nil
		if err := insertAudit(ctx, tx, auditRecord{
			eventType: "auth.account_unlocked", outcome: "success", reasonCode: "soft_lock_elapsed",
			actorUserID: &credential.userID, targetUserID: &credential.userID, metadata: metadata,
		}); err != nil {
			return CreatedSession{}, err
		}
	}

	passwordValid := false
	passwordReason := "invalid_password"
	if credential.hashScheme == "argon2id" && credential.passwordHash != "" {
		passwordValid, err = s.passwords.Verify(credential.passwordHash, request.Password)
		if err != nil {
			_, _ = s.passwords.Verify(s.dummyHash, request.Password)
			passwordReason = "unsupported_password_hash"
			passwordValid = false
		}
	} else {
		_, _ = s.passwords.Verify(s.dummyHash, request.Password)
		passwordReason = "unsupported_password_hash"
	}

	if !credential.userActive || !credential.credentialActive || credential.state != "active" || !passwordValid {
		reason := passwordReason
		if !credential.userActive || !credential.credentialActive {
			reason = "inactive_account"
		} else if credential.state != "active" {
			reason = "credential_" + credential.state
		}
		if credential.state == "active" && !passwordValid {
			credential.failedAttempts++
			newState := "active"
			var lockedUntil *time.Time
			if credential.failedAttempts >= s.config.LoginFailureLimit {
				newState = "soft_locked"
				deadline := now.Add(s.config.SoftLockDuration)
				lockedUntil = &deadline
			}
			if _, err := tx.Exec(ctx, `
				UPDATE user_credentials
				SET failed_attempts = $2, state = $3, soft_locked_until = $4,
					version = version + 1, updated_at = $5
				WHERE id = $1`, credential.credentialID, credential.failedAttempts, newState, lockedUntil, now); err != nil {
				return CreatedSession{}, fmt.Errorf("record failed login: %w", err)
			}
			if newState == "soft_locked" {
				if err := insertAudit(ctx, tx, auditRecord{
					eventType: "auth.account_locked", outcome: "failure", reasonCode: "failure_limit_reached",
					actorUserID: &credential.userID, targetUserID: &credential.userID, metadata: metadata,
				}); err != nil {
					return CreatedSession{}, err
				}
			}
		}
		if err := insertAudit(ctx, tx, auditRecord{
			eventType: "auth.login_failed", outcome: "failure", reasonCode: reason,
			actorUserID: &credential.userID, targetUserID: &credential.userID, metadata: metadata,
		}); err != nil {
			return CreatedSession{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return CreatedSession{}, fmt.Errorf("commit failed login: %w", err)
		}
		return CreatedSession{}, ErrAuthenticationFailed
	}

	contextValue, err := resolveContext(ctx, tx, credential.userID, request.InstitutionID, request.SiteID, request.FirmID)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return CreatedSession{}, fmt.Errorf("resolve login context: %w", err)
		}
		if err := insertAudit(ctx, tx, auditRecord{
			eventType: "auth.login_failed", outcome: "failure", reasonCode: "context_inaccessible",
			actorUserID: &credential.userID, targetUserID: &credential.userID, metadata: metadata,
		}); err != nil {
			return CreatedSession{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return CreatedSession{}, fmt.Errorf("commit inaccessible-context login: %w", err)
		}
		return CreatedSession{}, ErrAuthenticationFailed
	}

	permissions, err := loadPermissions(ctx, tx, credential.userID, request.InstitutionID)
	if err != nil {
		return CreatedSession{}, err
	}
	idleExpiresAt := now.Add(s.config.SessionIdleTimeout)
	absoluteExpiresAt := now.Add(s.config.SessionAbsoluteTimeout)
	var sessionID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO sessions (
			token_digest, csrf_digest, user_id, institution_id, site_id, firm_id,
			created_at, last_seen_at, idle_expires_at, absolute_expires_at,
			authorization_version
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $7, $8, $9, $10)
		RETURNING id`,
		tokenHash[:], csrfHash[:], credential.userID,
		contextValue.Institution.ID, contextValue.Site.ID, contextValue.Firm.ID,
		now, idleExpiresAt, absoluteExpiresAt, credential.authorizationVersion,
	).Scan(&sessionID); err != nil {
		return CreatedSession{}, fmt.Errorf("create session: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE user_credentials
		SET failed_attempts = 0, soft_locked_until = NULL, last_succeeded_at = $2,
			version = version + 1, updated_at = $2
		WHERE id = $1`, credential.credentialID, now); err != nil {
		return CreatedSession{}, fmt.Errorf("record successful login: %w", err)
	}
	if err := insertAudit(ctx, tx, auditRecord{
		eventType: "auth.login_succeeded", outcome: "success", reasonCode: "authenticated",
		actorUserID: &credential.userID, targetUserID: &credential.userID, sessionID: &sessionID,
		context: &contextValue, metadata: metadata,
	}); err != nil {
		return CreatedSession{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return CreatedSession{}, fmt.Errorf("commit login: %w", err)
	}

	return CreatedSession{
		Token: token,
		Session: Session{
			User:    User{ID: credential.userID, DisplayName: credential.displayName},
			Context: contextValue, Permissions: permissions, CSRFToken: csrf,
			IdleExpiresAt: idleExpiresAt, AbsoluteExpiresAt: absoluteExpiresAt, ContextVersion: 1,
		},
	}, nil
}

// CurrentSession validates and refreshes the current session.
func (s *Service) CurrentSession(ctx context.Context, token string, metadata RequestMetadata) (Session, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Session{}, fmt.Errorf("begin session transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	record, err := s.loadActiveSession(ctx, tx, token, metadata)
	if err != nil {
		if errors.Is(err, ErrUnauthenticated) {
			if commitErr := tx.Commit(ctx); commitErr != nil {
				return Session{}, fmt.Errorf("commit session rejection: %w", commitErr)
			}
		}
		return Session{}, err
	}
	permissions, err := loadPermissions(ctx, tx, record.user.ID, record.context.Institution.ID)
	if err != nil {
		return Session{}, err
	}
	if !slices.Contains(permissions, permissionReadSession) {
		if err := insertRateLimitedAudit(ctx, tx, record.audit("auth.authorization_denied", "denied", "missing_session_read_permission", metadata)); err != nil {
			return Session{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return Session{}, fmt.Errorf("commit session authorization denial: %w", err)
		}
		return Session{}, ErrForbidden
	}

	now := s.now().UTC()
	idleExpiry := record.idleExpiresAt
	if now.Sub(record.lastSeenAt) >= sessionRefreshInterval(s.config.SessionIdleTimeout) {
		idleExpiry = minTime(now.Add(s.config.SessionIdleTimeout), record.absoluteExpiresAt)
		if _, err := tx.Exec(ctx, `
			UPDATE sessions
			SET last_seen_at = $2, idle_expires_at = $3, version = version + 1
			WHERE id = $1`, record.id, now, idleExpiry); err != nil {
			return Session{}, fmt.Errorf("refresh session: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Session{}, fmt.Errorf("commit session refresh: %w", err)
	}
	return record.representation(token, s.config.CSRFKey, permissions, idleExpiry), nil
}

// AuthorizeOperation validates a session, CSRF token, live context, and permission.
// Denials are audited on a best-effort basis and always remain denials if audit is unavailable.
func (s *Service) AuthorizeOperation(ctx context.Context, request OperationAuthorizationRequest) (OperationPrincipal, error) {
	if strings.TrimSpace(request.Permission) == "" || strings.TrimSpace(request.DeniedEventType) == "" {
		return OperationPrincipal{}, errors.New("operation authorization policy is incomplete")
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return OperationPrincipal{}, fmt.Errorf("begin operation authorization transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	record, err := s.loadActiveSession(ctx, tx, request.Token, request.Metadata)
	if err != nil {
		if errors.Is(err, ErrUnauthenticated) {
			if commitErr := tx.Commit(ctx); commitErr != nil {
				return OperationPrincipal{}, fmt.Errorf("commit operation session rejection: %w", commitErr)
			}
		}
		return OperationPrincipal{}, err
	}
	if !verifyDigest(record.csrfDigest, request.CSRFToken) {
		s.bestEffortOperationDenial(ctx, tx, record, request, "csrf_rejected")
		return OperationPrincipal{}, ErrCSRF
	}

	firmID := record.context.Firm.ID
	validatedContext, err := resolveContext(
		ctx,
		tx,
		record.user.ID,
		record.context.Institution.ID,
		record.context.Site.ID,
		&firmID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.bestEffortOperationDenial(ctx, tx, record, request, "context_inaccessible")
			return OperationPrincipal{}, ErrForbidden
		}
		return OperationPrincipal{}, fmt.Errorf("validate operation context: %w", err)
	}

	permissions, err := loadPermissions(ctx, tx, record.user.ID, validatedContext.Institution.ID)
	if err != nil {
		return OperationPrincipal{}, err
	}
	if !slices.Contains(permissions, request.Permission) {
		s.bestEffortOperationDenial(ctx, tx, record, request, "permission_denied")
		return OperationPrincipal{}, ErrForbidden
	}

	now := s.now().UTC()
	if now.Sub(record.lastSeenAt) >= sessionRefreshInterval(s.config.SessionIdleTimeout) {
		idleExpiry := minTime(now.Add(s.config.SessionIdleTimeout), record.absoluteExpiresAt)
		if _, err := tx.Exec(ctx, `
			UPDATE sessions
			SET last_seen_at = $2, idle_expires_at = $3, version = version + 1
			WHERE id = $1`, record.id, now, idleExpiry); err != nil {
			return OperationPrincipal{}, fmt.Errorf("refresh authorized session: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return OperationPrincipal{}, fmt.Errorf("commit operation authorization: %w", err)
	}

	return OperationPrincipal{
		UserID:         record.user.ID,
		SessionID:      record.id,
		InstitutionID:  validatedContext.Institution.ID,
		SiteID:         validatedContext.Site.ID,
		FirmID:         validatedContext.Firm.ID,
		ContextVersion: record.contextVersion,
	}, nil
}

func (s *Service) bestEffortOperationDenial(
	ctx context.Context,
	tx pgx.Tx,
	record sessionRecord,
	request OperationAuthorizationRequest,
	reason string,
) {
	denial := record.audit(request.DeniedEventType, "denied", reason, request.Metadata)
	if err := insertRateLimitedAudit(ctx, tx, denial); err != nil {
		return
	}
	_ = tx.Commit(ctx)
}

// Logout revokes only the current session after CSRF and permission checks.
func (s *Service) Logout(ctx context.Context, token, csrf string, metadata RequestMetadata) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin logout transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	record, err := s.loadActiveSession(ctx, tx, token, metadata)
	if err != nil {
		if errors.Is(err, ErrUnauthenticated) {
			if commitErr := tx.Commit(ctx); commitErr != nil {
				return fmt.Errorf("commit logout rejection: %w", commitErr)
			}
		}
		return err
	}
	if !verifyDigest(record.csrfDigest, csrf) {
		if err := insertRateLimitedAudit(ctx, tx, record.audit("auth.csrf_rejected", "denied", "csrf_mismatch", metadata)); err != nil {
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit CSRF rejection: %w", err)
		}
		return ErrCSRF
	}
	permissions, err := loadPermissions(ctx, tx, record.user.ID, record.context.Institution.ID)
	if err != nil {
		return err
	}
	if !slices.Contains(permissions, permissionRevokeSession) {
		if err := insertRateLimitedAudit(ctx, tx, record.audit("auth.authorization_denied", "denied", "missing_session_revoke_permission", metadata)); err != nil {
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit logout authorization denial: %w", err)
		}
		return ErrForbidden
	}

	now := s.now().UTC()
	if _, err := tx.Exec(ctx, `
		UPDATE sessions SET revoked_at = $2, revocation_reason = 'logout', version = version + 1
		WHERE id = $1 AND revoked_at IS NULL`, record.id, now); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	if err := insertAudit(ctx, tx, record.audit("auth.logout", "success", "user_logout", metadata)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit logout: %w", err)
	}
	return nil
}

// ReplaceContext atomically replaces the complete session context.
func (s *Service) ReplaceContext(ctx context.Context, token, csrf string, request ContextRequest, metadata RequestMetadata) (Session, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Session{}, fmt.Errorf("begin context transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	record, err := s.loadActiveSession(ctx, tx, token, metadata)
	if err != nil {
		if errors.Is(err, ErrUnauthenticated) {
			if commitErr := tx.Commit(ctx); commitErr != nil {
				return Session{}, fmt.Errorf("commit context rejection: %w", commitErr)
			}
		}
		return Session{}, err
	}
	if !verifyDigest(record.csrfDigest, csrf) {
		if err := insertRateLimitedAudit(ctx, tx, record.audit("auth.csrf_rejected", "denied", "csrf_mismatch", metadata)); err != nil {
			return Session{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return Session{}, fmt.Errorf("commit CSRF rejection: %w", err)
		}
		return Session{}, ErrCSRF
	}
	permissions, err := loadPermissions(ctx, tx, record.user.ID, record.context.Institution.ID)
	if err != nil {
		return Session{}, err
	}
	if !slices.Contains(permissions, permissionSwitchContext) {
		if err := insertRateLimitedAudit(ctx, tx, record.audit("auth.authorization_denied", "denied", "missing_context_switch_permission", metadata)); err != nil {
			return Session{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return Session{}, fmt.Errorf("commit context authorization denial: %w", err)
		}
		return Session{}, ErrForbidden
	}
	if request.Version != record.contextVersion {
		return Session{}, ErrConflict
	}

	requestedFirm := request.FirmID
	newContext, err := resolveContext(ctx, tx, record.user.ID, request.InstitutionID, request.SiteID, &requestedFirm)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return Session{}, fmt.Errorf("resolve replacement context: %w", err)
		}
		if err := insertRateLimitedAudit(ctx, tx, record.audit("auth.authorization_denied", "denied", "context_inaccessible", metadata)); err != nil {
			return Session{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return Session{}, fmt.Errorf("commit context denial: %w", err)
		}
		return Session{}, ErrForbidden
	}

	newVersion := record.contextVersion + 1
	command, err := tx.Exec(ctx, `
		UPDATE sessions
		SET institution_id = $2, site_id = $3, firm_id = $4,
			context_version = $5, version = version + 1
		WHERE id = $1 AND context_version = $6`,
		record.id, newContext.Institution.ID, newContext.Site.ID, newContext.Firm.ID,
		newVersion, record.contextVersion,
	)
	if err != nil {
		return Session{}, fmt.Errorf("replace session context: %w", err)
	}
	if command.RowsAffected() != 1 {
		return Session{}, ErrConflict
	}
	if err := insertAudit(ctx, tx, auditRecord{
		eventType: "auth.context_changed", outcome: "success", reasonCode: "context_replaced",
		actorUserID: &record.user.ID, targetUserID: &record.user.ID, sessionID: &record.id,
		context: &newContext, metadata: metadata,
	}); err != nil {
		return Session{}, err
	}
	permissions, err = loadPermissions(ctx, tx, record.user.ID, newContext.Institution.ID)
	if err != nil {
		return Session{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Session{}, fmt.Errorf("commit context replacement: %w", err)
	}

	record.context = newContext
	record.contextVersion = newVersion
	return record.representation(token, s.config.CSRFKey, permissions, record.idleExpiresAt), nil
}

type loginCredential struct {
	credentialID         int64
	userID               int64
	displayName          string
	passwordHash         string
	hashScheme           string
	state                string
	failedAttempts       int
	softLockedUntil      *time.Time
	credentialActive     bool
	userActive           bool
	authorizationVersion int64
}

func loadCredentialForLogin(ctx context.Context, tx pgx.Tx, username string, institutionID int64) (loginCredential, error) {
	var value loginCredential
	err := tx.QueryRow(ctx, `
		SELECT c.id, c.user_id, u.display_name, COALESCE(c.password_hash, ''),
			COALESCE(c.hash_scheme, ''), c.state::text, c.failed_attempts,
			c.soft_locked_until, c.active, u.active, u.authorization_version
		FROM user_credentials c
		JOIN users u ON u.id = c.user_id
		JOIN authentication_profiles p ON p.id = c.authentication_profile_id
		WHERE c.canonical_username = $1
			AND p.method = 'LOCAL' AND p.active
			AND p.institution_id = $2
		FOR UPDATE OF c`, username, institutionID).Scan(
		&value.credentialID, &value.userID, &value.displayName, &value.passwordHash,
		&value.hashScheme, &value.state, &value.failedAttempts, &value.softLockedUntil,
		&value.credentialActive, &value.userActive, &value.authorizationVersion,
	)
	return value, err
}

func resolveContext(ctx context.Context, tx pgx.Tx, userID, institutionID, siteID int64, firmID *int64) (UserContext, error) {
	var value UserContext
	if err := tx.QueryRow(ctx, `
		SELECT i.id, i.name, s.id, s.name
		FROM institutions i
		JOIN user_institution_memberships uim ON uim.institution_id = i.id AND uim.user_id = $1 AND uim.active
		JOIN sites s ON s.id = $3 AND s.institution_id = i.id AND s.active
		JOIN user_site_memberships usm ON usm.site_id = s.id AND usm.user_id = $1 AND usm.active
		WHERE i.id = $2 AND i.active`, userID, institutionID, siteID).Scan(
		&value.Institution.ID, &value.Institution.Name, &value.Site.ID, &value.Site.Name,
	); err != nil {
		return UserContext{}, err
	}

	if firmID != nil {
		err := tx.QueryRow(ctx, `
			SELECT f.id, f.name
			FROM firms f
			JOIN user_firm_memberships ufm ON ufm.firm_id = f.id AND ufm.user_id = $1 AND ufm.active
			WHERE f.id = $3 AND f.active
				AND (f.institution_id = $2 OR (f.global_access AND f.institution_id IS NULL))`,
			userID, institutionID, *firmID).Scan(&value.Firm.ID, &value.Firm.Name)
		if err != nil {
			return UserContext{}, err
		}
		return value, nil
	}

	err := tx.QueryRow(ctx, `
		SELECT f.id, f.name
		FROM firms f
		JOIN user_firm_memberships ufm ON ufm.firm_id = f.id AND ufm.user_id = $1 AND ufm.active
		LEFT JOIN user_firm_preferences ufp ON ufp.firm_id = f.id AND ufp.user_id = $1
		WHERE f.active AND (f.institution_id = $2 OR (f.global_access AND f.institution_id IS NULL))
		ORDER BY ufp.position NULLS LAST, f.id
		LIMIT 1`, userID, institutionID).Scan(&value.Firm.ID, &value.Firm.Name)
	if err != nil {
		return UserContext{}, err
	}
	return value, nil
}

func loadPermissions(ctx context.Context, tx pgx.Tx, userID, institutionID int64) ([]string, error) {
	rows, err := tx.Query(ctx, `
		SELECT DISTINCT p.name
		FROM user_role_assignments ura
		JOIN roles r ON r.id = ura.role_id AND r.active
		JOIN role_permissions rp ON rp.role_id = r.id AND rp.active
		JOIN permissions p ON p.id = rp.permission_id AND p.active
		WHERE ura.user_id = $1 AND ura.active
			AND (ura.institution_id = $2 OR ura.institution_id IS NULL)
		ORDER BY p.name`, userID, institutionID)
	if err != nil {
		return nil, fmt.Errorf("load permissions: %w", err)
	}
	defer rows.Close()

	permissions := make([]string, 0)
	for rows.Next() {
		var permission string
		if err := rows.Scan(&permission); err != nil {
			return nil, fmt.Errorf("scan permission: %w", err)
		}
		permissions = append(permissions, permission)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate permissions: %w", err)
	}
	return permissions, nil
}

type sessionRecord struct {
	id                   int64
	user                 User
	userActive           bool
	context              UserContext
	csrfDigest           []byte
	createdAt            time.Time
	lastSeenAt           time.Time
	idleExpiresAt        time.Time
	absoluteExpiresAt    time.Time
	revokedAt            *time.Time
	authorizationVersion int64
	currentAuthzVersion  int64
	contextVersion       int64
}

func (s *Service) loadActiveSession(ctx context.Context, tx pgx.Tx, token string, metadata RequestMetadata) (sessionRecord, error) {
	hash := tokenDigest(token)
	var value sessionRecord
	err := tx.QueryRow(ctx, `
		SELECT s.id, s.user_id, u.display_name, u.active,
			i.id, i.name, st.id, st.name, f.id, f.name,
			s.csrf_digest, s.created_at, s.last_seen_at, s.idle_expires_at, s.absolute_expires_at,
			s.revoked_at, s.authorization_version, u.authorization_version, s.context_version
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		JOIN institutions i ON i.id = s.institution_id
		JOIN sites st ON st.id = s.site_id
		JOIN firms f ON f.id = s.firm_id
		WHERE s.token_digest = $1
		FOR UPDATE OF s`, hash[:]).Scan(
		&value.id, &value.user.ID, &value.user.DisplayName, &value.userActive,
		&value.context.Institution.ID, &value.context.Institution.Name,
		&value.context.Site.ID, &value.context.Site.Name,
		&value.context.Firm.ID, &value.context.Firm.Name,
		&value.csrfDigest, &value.createdAt, &value.lastSeenAt, &value.idleExpiresAt, &value.absoluteExpiresAt,
		&value.revokedAt, &value.authorizationVersion, &value.currentAuthzVersion, &value.contextVersion,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sessionRecord{}, ErrUnauthenticated
		}
		return sessionRecord{}, fmt.Errorf("load session: %w", err)
	}
	if value.revokedAt != nil {
		return sessionRecord{}, ErrUnauthenticated
	}

	now := s.now().UTC()
	if !value.userActive {
		if _, err := tx.Exec(ctx, `
			UPDATE sessions SET revoked_at = $2, revocation_reason = 'account_disabled', version = version + 1
			WHERE id = $1 AND revoked_at IS NULL`, value.id, now); err != nil {
			return sessionRecord{}, fmt.Errorf("revoke inactive-user session: %w", err)
		}
		if err := insertAudit(ctx, tx, value.audit("auth.session_revoked", "failure", "account_disabled", metadata)); err != nil {
			return sessionRecord{}, err
		}
		return sessionRecord{}, ErrUnauthenticated
	}
	reason := ""
	if !now.Before(value.absoluteExpiresAt) {
		reason = "absolute_expired"
	} else if !now.Before(value.idleExpiresAt) {
		reason = "idle_expired"
	} else if value.authorizationVersion != value.currentAuthzVersion {
		reason = "authorization_changed"
	}
	if reason != "" {
		if _, err := tx.Exec(ctx, `
			UPDATE sessions SET revoked_at = $2, revocation_reason = $3, version = version + 1
			WHERE id = $1 AND revoked_at IS NULL`, value.id, now, reason); err != nil {
			return sessionRecord{}, fmt.Errorf("expire session: %w", err)
		}
		eventType := "auth.session_expired"
		if reason == "authorization_changed" {
			eventType = "auth.session_revoked"
		}
		if err := insertAudit(ctx, tx, value.audit(eventType, "failure", reason, metadata)); err != nil {
			return sessionRecord{}, err
		}
		return sessionRecord{}, ErrUnauthenticated
	}
	return value, nil
}

func (r sessionRecord) representation(token string, key []byte, permissions []string, idleExpiry time.Time) Session {
	return Session{
		User: r.user, Context: r.context, Permissions: permissions,
		CSRFToken: csrfToken(key, token), IdleExpiresAt: idleExpiry,
		AbsoluteExpiresAt: r.absoluteExpiresAt, ContextVersion: r.contextVersion,
	}
}

func (r sessionRecord) audit(eventType, outcome, reason string, metadata RequestMetadata) auditRecord {
	return auditRecord{
		eventType: eventType, outcome: outcome, reasonCode: reason,
		actorUserID: &r.user.ID, targetUserID: &r.user.ID, sessionID: &r.id,
		context: &r.context, metadata: metadata,
	}
}

type auditRecord struct {
	eventType    string
	outcome      string
	reasonCode   string
	actorUserID  *int64
	targetUserID *int64
	sessionID    *int64
	context      *UserContext
	metadata     RequestMetadata
}

func insertAudit(ctx context.Context, tx pgx.Tx, record auditRecord) error {
	var institutionID, siteID, firmID *int64
	if record.context != nil {
		institutionID = &record.context.Institution.ID
		siteID = &record.context.Site.ID
		firmID = &record.context.Firm.ID
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (
			event_type, actor_user_id, target_user_id, session_id,
			institution_id, site_id, firm_id, outcome, reason_code,
			correlation_id, source_ip_class
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		record.eventType, record.actorUserID, record.targetUserID, record.sessionID,
		institutionID, siteID, firmID, record.outcome, record.reasonCode,
		record.metadata.CorrelationID, record.metadata.SourceIPClass,
	); err != nil {
		return fmt.Errorf("append audit event: %w", err)
	}
	return nil
}

func insertRateLimitedAudit(ctx context.Context, tx pgx.Tx, record auditRecord) error {
	if record.sessionID == nil {
		return errors.New("rate-limited audit requires a session")
	}
	lockKey := fmt.Sprintf("%d:%s:%s", *record.sessionID, record.eventType, record.reasonCode)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, lockKey); err != nil {
		return fmt.Errorf("lock rate-limited audit: %w", err)
	}

	var institutionID, siteID, firmID *int64
	if record.context != nil {
		institutionID = &record.context.Institution.ID
		siteID = &record.context.Site.ID
		firmID = &record.context.Firm.ID
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (
			event_type, actor_user_id, target_user_id, session_id,
			institution_id, site_id, firm_id, outcome, reason_code,
			correlation_id, source_ip_class
		)
		SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		WHERE NOT EXISTS (
			SELECT 1 FROM audit_events
			WHERE session_id = $4 AND event_type = $1 AND reason_code = $9
				AND occurred_at >= now() - interval '1 minute'
		)`,
		record.eventType, record.actorUserID, record.targetUserID, record.sessionID,
		institutionID, siteID, firmID, record.outcome, record.reasonCode,
		record.metadata.CorrelationID, record.metadata.SourceIPClass,
	); err != nil {
		return fmt.Errorf("append rate-limited audit event: %w", err)
	}
	return nil
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func sessionRefreshInterval(idleTimeout time.Duration) time.Duration {
	interval := idleTimeout / 4
	if interval <= 0 {
		return idleTimeout
	}
	return min(interval, time.Minute)
}
