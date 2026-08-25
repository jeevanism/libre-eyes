package admin

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jeevanism/visionopus/internal/auth"
)

type authorizer interface {
	AuthorizeOperation(context.Context, auth.OperationAuthorizationRequest) (auth.OperationPrincipal, error)
	AuthorizeRead(context.Context, auth.ReadAuthorizationRequest) (auth.OperationPrincipal, error)
}
type Authorization struct {
	principal auth.OperationPrincipal
	metadata  auth.RequestMetadata
	service   *Service
}
type Service struct {
	pool       *pgxpool.Pool
	authorizer authorizer
}

func NewService(pool *pgxpool.Pool, a authorizer) (*Service, error) {
	if pool == nil || a == nil {
		return nil, errors.New("admin database and authorizer are required")
	}
	return &Service{pool: pool, authorizer: a}, nil
}
func (s *Service) Authorize(ctx context.Context, token, csrf string, metadata auth.RequestMetadata, write bool) (Authorization, error) {
	var p auth.OperationPrincipal
	var err error
	if write {
		p, err = s.authorizer.AuthorizeOperation(ctx, auth.OperationAuthorizationRequest{Token: token, CSRFToken: csrf, Permission: PermissionManage, DeniedEventType: "admin.denied", Metadata: metadata})
	} else {
		p, err = s.authorizer.AuthorizeRead(ctx, auth.ReadAuthorizationRequest{Token: token, Permission: PermissionRead, DeniedEventType: "admin.denied", Metadata: metadata})
	}
	if err != nil {
		return Authorization{}, err
	}
	// Keep the administration boundary explicit in addition to the generic
	// permission check. This prevents stale or over-broad role permissions
	// from granting clinical users access to the admin workspace.
	var role string
	if err := s.pool.QueryRow(ctx, `
		SELECT role_code
		FROM development_admin_users
		WHERE user_id=$1 AND institution_id=$2 AND active
		  AND role_code IN ('system_administrator','institution_administrator')
		LIMIT 1`, p.UserID, p.InstitutionID).Scan(&role); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Authorization{}, ErrForbidden
		}
		return Authorization{}, fmt.Errorf("check administration role: %w", err)
	}
	return Authorization{principal: p, metadata: metadata, service: s}, nil
}
func (s *Service) Users(ctx context.Context, a Authorization) ([]User, error) {
	rows, err := s.pool.Query(ctx, `SELECT public_id::text,username,display_name,role_code,active,version FROM development_admin_users WHERE institution_id=$1 ORDER BY display_name`, a.principal.InstitutionID)
	if err != nil {
		return nil, fmt.Errorf("list admin users: %w", err)
	}
	defer rows.Close()
	out := []User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Role, &u.Active, &u.Version); err != nil {
			return nil, err
		}
		if err := s.loadUserDetails(ctx, a, &u); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Service) loadUserDetails(ctx context.Context, a Authorization, u *User) error {
	u.Permissions = []string{}
	u.Sites = []Reference{}
	u.Firms = []Reference{}
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT p.name
		FROM user_role_assignments ura
		JOIN roles r ON r.id=ura.role_id AND r.scope=ura.role_scope AND r.active
		JOIN role_permissions rp ON rp.role_id=r.id AND rp.active
		JOIN permissions p ON p.id=rp.permission_id AND p.active
		WHERE ura.user_id=(SELECT user_id FROM development_admin_users WHERE public_id=$1::uuid)
		  AND ura.active AND (ura.institution_id=$2 OR ura.institution_id IS NULL)
		  AND (p.name NOT LIKE 'admin.development.%' OR EXISTS (
			SELECT 1 FROM development_admin_users dau
			WHERE dau.user_id=ura.user_id AND dau.institution_id=$2 AND dau.active
			  AND dau.role_code IN ('system_administrator','institution_administrator')
		  ))
		ORDER BY p.name`, u.ID, a.principal.InstitutionID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			rows.Close()
			return err
		}
		u.Permissions = append(u.Permissions, p)
	}
	rows.Close()
	rows, err = s.pool.Query(ctx, `SELECT s.id,s.name FROM user_site_memberships m JOIN sites s ON s.id=m.site_id WHERE m.user_id=(SELECT user_id FROM development_admin_users WHERE public_id=$1::uuid) AND m.active AND s.institution_id=$2 AND s.active ORDER BY s.name`, u.ID, a.principal.InstitutionID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var v Reference
		if err := rows.Scan(&v.ID, &v.Name); err != nil {
			rows.Close()
			return err
		}
		u.Sites = append(u.Sites, v)
	}
	rows.Close()
	rows, err = s.pool.Query(ctx, `SELECT f.id,f.name FROM user_firm_memberships m JOIN firms f ON f.id=m.firm_id WHERE m.user_id=(SELECT user_id FROM development_admin_users WHERE public_id=$1::uuid) AND m.active AND (f.institution_id=$2 OR f.global_access) AND f.active ORDER BY f.name`, u.ID, a.principal.InstitutionID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var v Reference
		if err := rows.Scan(&v.ID, &v.Name); err != nil {
			return err
		}
		u.Firms = append(u.Firms, v)
	}
	return rows.Err()
}
func (s *Service) Contexts(ctx context.Context, a Authorization) (Contexts, error) {
	var out Contexts
	if err := s.pool.QueryRow(ctx, `SELECT id,name FROM institutions WHERE id=$1`, a.principal.InstitutionID).Scan(&out.Institution.ID, &out.Institution.Name); err != nil {
		return out, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id,name,active,version FROM sites WHERE institution_id=$1 ORDER BY name`, a.principal.InstitutionID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var r Reference
		if err := rows.Scan(&r.ID, &r.Name, &r.Active, &r.Version); err != nil {
			return out, err
		}
		out.Sites = append(out.Sites, r)
	}
	rows, err = s.pool.Query(ctx, `SELECT id,name,active,version FROM firms WHERE institution_id=$1 ORDER BY name`, a.principal.InstitutionID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var r Reference
		if err := rows.Scan(&r.ID, &r.Name, &r.Active, &r.Version); err != nil {
			return out, err
		}
		out.Firms = append(out.Firms, r)
	}
	return out, rows.Err()
}
func (s *Service) Settings(ctx context.Context, a Authorization) ([]Setting, error) {
	rows, err := s.pool.Query(ctx, `SELECT key,value,version FROM development_admin_settings WHERE institution_id=$1 ORDER BY key`, a.principal.InstitutionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Setting{}
	for rows.Next() {
		var v Setting
		if err := rows.Scan(&v.Key, &v.Value, &v.Version); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Service) Audit(ctx context.Context, a Authorization) ([]AuditEvent, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT a.actor_user_id, u.display_name, a.command, a.target_type,
		       a.target_public_id::text, a.target_key, target_user.display_name,
		       a.changed_fields, a.outcome,
		       a.correlation_id, a.created_at
		FROM development_admin_audit a
		JOIN users u ON u.id=a.actor_user_id
		LEFT JOIN development_admin_users target_admin ON target_admin.public_id=a.target_public_id
		LEFT JOIN users target_user ON target_user.id=target_admin.user_id
		WHERE a.institution_id=$1
		ORDER BY a.created_at DESC,a.id DESC
		LIMIT 100`, a.principal.InstitutionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AuditEvent{}
	for rows.Next() {
		var v AuditEvent
		var raw []byte
		if err := rows.Scan(&v.ActorUserID, &v.ActorDisplayName, &v.Command, &v.TargetType, &v.TargetPublicID, &v.TargetKey, &v.TargetDisplayName, &raw, &v.Outcome, &v.CorrelationID, &v.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(raw, &v.ChangedFields)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Service) SetUserActive(ctx context.Context, a Authorization, cmd UserCommand) (User, error) {
	if cmd.PublicID == "" || cmd.ExpectedVersion < 1 {
		return User{}, ErrInvalidRequest
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var u User
	var userID int64
	err = tx.QueryRow(ctx, `UPDATE development_admin_users SET active=$1, version=version+1, updated_at=now() WHERE public_id=$2::uuid AND institution_id=$3 AND version=$4 RETURNING user_id,public_id::text,username,display_name,role_code,active,version`, cmd.Active, cmd.PublicID, a.principal.InstitutionID, cmd.ExpectedVersion).Scan(&userID, &u.ID, &u.Username, &u.DisplayName, &u.Role, &u.Active, &u.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrConflict
	}
	if err != nil {
		return User{}, err
	}
	if _, err = tx.Exec(ctx, `
		UPDATE users
		SET active=$1, authorization_version=authorization_version+1, updated_at=now()
		WHERE id=$2`, cmd.Active, userID); err != nil {
		return User{}, fmt.Errorf("update user authentication state: %w", err)
	}
	credentialState := "disabled"
	if cmd.Active {
		credentialState = "active"
	}
	if _, err = tx.Exec(ctx, `
		UPDATE user_credentials
		SET active=$1, state=$2::credential_state, updated_at=now(), version=version+1
		WHERE user_id=$3`, cmd.Active, credentialState, userID); err != nil {
		return User{}, fmt.Errorf("update user credentials state: %w", err)
	}
	if _, err = tx.Exec(ctx, `
		UPDATE sessions
		SET revoked_at=now(), revocation_reason='account_disabled', version=version+1
		WHERE user_id=$1 AND revoked_at IS NULL AND NOT $2`, userID, cmd.Active); err != nil {
		return User{}, fmt.Errorf("revoke disabled-user sessions: %w", err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO development_admin_audit(actor_user_id,institution_id,command,target_type,target_public_id,changed_fields,outcome,correlation_id) VALUES($1,$2,$3,'user',$4::uuid,'["active"]'::jsonb,'success',$5)`, a.principal.UserID, a.principal.InstitutionID, "user.active", cmd.PublicID, a.metadata.CorrelationID); err != nil {
		return User{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return u, nil
}

func (s *Service) CreateUser(ctx context.Context, a Authorization, in UserUpsert) (User, error) {
	if err := validateUserInput(in, false); err != nil || in.Password == "" {
		return User{}, ErrInvalidRequest
	}
	hash, err := (auth.PasswordManager{}).Hash(in.Password)
	if err != nil {
		return User{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := validateMemberships(ctx, tx, a.principal.InstitutionID, in.SiteIDs, in.FirmIDs); err != nil {
		return User{}, err
	}
	var userID, profileID int64
	if err := tx.QueryRow(ctx, `INSERT INTO users(display_name,created_by_user_id,updated_by_user_id) VALUES($1,$2,$2) RETURNING id`, in.DisplayName, a.principal.UserID).Scan(&userID); err != nil {
		return User{}, err
	}
	if err := tx.QueryRow(ctx, `SELECT id FROM authentication_profiles WHERE institution_id=$1 AND method='LOCAL' AND active ORDER BY id LIMIT 1`, a.principal.InstitutionID).Scan(&profileID); err != nil {
		return User{}, err
	}
	username := strings.ToLower(strings.TrimSpace(in.Username))
	if _, err := tx.Exec(ctx, `INSERT INTO user_credentials(user_id,authentication_profile_id,canonical_username,password_hash,hash_scheme,hash_version,password_changed_at) VALUES($1,$2,$3,$4,'argon2id',19,now())`, userID, profileID, username, hash); err != nil {
		return User{}, err
	}
	if err := assignUserContext(ctx, tx, a.principal.InstitutionID, a.principal.UserID, userID, in.Role, in.SiteIDs, in.FirmIDs); err != nil {
		return User{}, err
	}
	publicID, err := newPublicID()
	if err != nil {
		return User{}, err
	}
	var u User
	if err := tx.QueryRow(ctx, `INSERT INTO development_admin_users(public_id,institution_id,user_id,username,display_name,role_code) VALUES($1::uuid,$2,$3,$4,$5,$6) RETURNING public_id::text,username,display_name,role_code,active,version`, publicID, a.principal.InstitutionID, userID, username, in.DisplayName, in.Role).Scan(&u.ID, &u.Username, &u.DisplayName, &u.Role, &u.Active, &u.Version); err != nil {
		return User{}, err
	}
	if err := auditTx(ctx, tx, a, "user.create", "user", publicID, `["username","display_name","role","context"]`); err != nil {
		return User{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}
	if err := s.loadUserDetails(ctx, a, &u); err != nil {
		return User{}, err
	}
	return u, nil
}

func (s *Service) UpdateUser(ctx context.Context, a Authorization, in UserUpsert) (User, error) {
	if in.PublicID == "" || validateUserInput(in, true) != nil {
		return User{}, ErrInvalidRequest
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := validateMemberships(ctx, tx, a.principal.InstitutionID, in.SiteIDs, in.FirmIDs); err != nil {
		return User{}, err
	}
	var userID int64
	var u User
	err = tx.QueryRow(ctx, `UPDATE development_admin_users SET username=$1,display_name=$2,role_code=$3,version=version+1,updated_at=now() WHERE public_id=$4::uuid AND institution_id=$5 AND version=$6 RETURNING user_id,public_id::text,username,display_name,role_code,active,version`, strings.ToLower(strings.TrimSpace(in.Username)), strings.TrimSpace(in.DisplayName), in.Role, in.PublicID, a.principal.InstitutionID, in.ExpectedVersion).Scan(&userID, &u.ID, &u.Username, &u.DisplayName, &u.Role, &u.Active, &u.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrConflict
	}
	if err != nil {
		return User{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE users SET display_name=$1,updated_by_user_id=$2,updated_at=now() WHERE id=$3`, in.DisplayName, a.principal.UserID, userID); err != nil {
		return User{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE user_credentials SET canonical_username=$1,updated_at=now(),version=version+1 WHERE user_id=$2`, strings.ToLower(strings.TrimSpace(in.Username)), userID); err != nil {
		return User{}, err
	}
	if in.Password != "" {
		if len(in.Password) < 6 {
			return User{}, ErrInvalidRequest
		}
		hash, e := (auth.PasswordManager{}).Hash(in.Password)
		if e != nil {
			return User{}, e
		}
		if _, e = tx.Exec(ctx, `UPDATE user_credentials SET password_hash=$1,hash_scheme='argon2id',hash_version=19,password_changed_at=now(),state='active',active=TRUE,failed_attempts=0,soft_locked_until=NULL,version=version+1,updated_at=now() WHERE user_id=$2`, hash, userID); e != nil {
			return User{}, e
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE user_role_assignments ura SET active=FALSE FROM roles r WHERE ura.role_id=r.id AND ura.role_scope=r.scope AND ura.user_id=$1 AND ura.institution_id=$2 AND r.scope='institution'`, userID, a.principal.InstitutionID); err != nil {
		return User{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE user_site_memberships m SET active=FALSE FROM sites s WHERE m.site_id=s.id AND m.user_id=$1 AND s.institution_id=$2`, userID, a.principal.InstitutionID); err != nil {
		return User{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE user_firm_memberships m SET active=FALSE FROM firms f WHERE m.firm_id=f.id AND m.user_id=$1 AND (f.institution_id=$2 OR f.global_access)`, userID, a.principal.InstitutionID); err != nil {
		return User{}, err
	}
	if err = assignUserContext(ctx, tx, a.principal.InstitutionID, a.principal.UserID, userID, in.Role, in.SiteIDs, in.FirmIDs); err != nil {
		return User{}, err
	}
	if err = auditTx(ctx, tx, a, "user.update", "user", in.PublicID, `["username","display_name","role","context"]`); err != nil {
		return User{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return User{}, err
	}
	if err = s.loadUserDetails(ctx, a, &u); err != nil {
		return User{}, err
	}
	return u, nil
}

func validateUserInput(in UserUpsert, update bool) error {
	if update && in.ExpectedVersion < 1 {
		return ErrInvalidRequest
	}
	username := strings.TrimSpace(in.Username)
	if username == "" || len(username) > 80 || username != strings.ToLower(username) || strings.ContainsAny(username, " \t\r\n") {
		return ErrInvalidRequest
	}
	if strings.TrimSpace(in.DisplayName) == "" || len(strings.TrimSpace(in.DisplayName)) > 120 {
		return ErrInvalidRequest
	}
	if in.Role != "clinical_user" && in.Role != "institution_administrator" {
		return ErrInvalidRequest
	}
	if !update && len(in.Password) < 6 {
		return ErrInvalidRequest
	}
	return nil
}

func validateMemberships(ctx context.Context, tx pgx.Tx, institutionID int64, siteIDs, firmIDs []int64) error {
	for _, id := range siteIDs {
		var ok bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM sites WHERE id=$1 AND institution_id=$2 AND active)`, id, institutionID).Scan(&ok); err != nil || !ok {
			return ErrInvalidRequest
		}
	}
	for _, id := range firmIDs {
		var ok bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM firms WHERE id=$1 AND (institution_id=$2 OR global_access) AND active)`, id, institutionID).Scan(&ok); err != nil || !ok {
			return ErrInvalidRequest
		}
	}
	return nil
}

func assignUserContext(ctx context.Context, tx pgx.Tx, institutionID, actorID, userID int64, role string, siteIDs, firmIDs []int64) error {
	if _, err := tx.Exec(ctx, `INSERT INTO user_institution_memberships(user_id,institution_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, userID, institutionID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO user_role_assignments(user_id,role_id,role_scope,institution_id,assigned_by_user_id) SELECT $1,id,scope,$2,$3 FROM roles WHERE name IN ('VisionOpus User','Development Patient Search Tester') AND scope='institution' ON CONFLICT (user_id,role_id,institution_id) DO UPDATE SET active=TRUE`, userID, institutionID, actorID); err != nil {
		return err
	}
	if role == "institution_administrator" {
		if _, err := tx.Exec(ctx, `INSERT INTO user_role_assignments(user_id,role_id,role_scope,institution_id,assigned_by_user_id) SELECT $1,id,scope,$2,$3 FROM roles WHERE name='Institution Administrator' AND scope='institution' ON CONFLICT (user_id,role_id,institution_id) DO UPDATE SET active=TRUE`, userID, institutionID, actorID); err != nil {
			return err
		}
	}
	for _, id := range siteIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO user_site_memberships(user_id,site_id) VALUES($1,$2) ON CONFLICT (user_id,site_id) DO UPDATE SET active=TRUE`, userID, id); err != nil {
			return err
		}
	}
	for _, id := range firmIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO user_firm_memberships(user_id,firm_id) VALUES($1,$2) ON CONFLICT (user_id,firm_id) DO UPDATE SET active=TRUE`, userID, id); err != nil {
			return err
		}
	}
	return nil
}

func newPublicID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

func auditTx(ctx context.Context, tx pgx.Tx, a Authorization, command, targetType, targetID, fields string) error {
	_, err := tx.Exec(ctx, `INSERT INTO development_admin_audit(actor_user_id,institution_id,command,target_type,target_public_id,changed_fields,outcome,correlation_id) VALUES($1,$2,$3,$4,$5::uuid,$6::jsonb,'success',$7)`, a.principal.UserID, a.principal.InstitutionID, command, targetType, targetID, fields, a.metadata.CorrelationID)
	return err
}

func (s *Service) UpdateSetting(ctx context.Context, a Authorization, in SettingUpdate) (Setting, error) {
	if in.ExpectedVersion < 1 || !validSettingValue(in.Key, in.Value) {
		return Setting{}, ErrInvalidRequest
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Setting{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := validateDefaultContext(ctx, tx, a.principal.InstitutionID, in.Key, in.Value); err != nil {
		return Setting{}, err
	}
	var out Setting
	err = tx.QueryRow(ctx, `UPDATE development_admin_settings SET value=$1,version=version+1,updated_at=now() WHERE key=$2 AND institution_id=$3 AND version=$4 RETURNING key,value,version`, in.Value, in.Key, a.principal.InstitutionID, in.ExpectedVersion).Scan(&out.Key, &out.Value, &out.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return Setting{}, ErrConflict
	}
	if err != nil {
		return Setting{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO development_admin_audit(actor_user_id,institution_id,command,target_type,target_key,changed_fields,outcome,correlation_id) VALUES($1,$2,'setting.update','setting',$3,'["value"]'::jsonb,'success',$4)`, a.principal.UserID, a.principal.InstitutionID, in.Key, a.metadata.CorrelationID); err != nil {
		return Setting{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Setting{}, err
	}
	return out, nil
}

func validateDefaultContext(ctx context.Context, tx pgx.Tx, institutionID int64, key, value string) error {
	if key != "default_site" && key != "default_firm" {
		return nil
	}
	table := "sites"
	if key == "default_firm" {
		table = "firms"
	}
	var exists bool
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE institution_id=$1 AND name=$2 AND active)", table)
	if err := tx.QueryRow(ctx, query, institutionID, strings.TrimSpace(value)).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrInvalidRequest
	}
	return nil
}

func (s *Service) UpsertSite(ctx context.Context, a Authorization, in ContextUpsert) (Reference, error) {
	return s.upsertContext(ctx, a, in, false)
}

func (s *Service) UpsertFirm(ctx context.Context, a Authorization, in ContextUpsert) (Reference, error) {
	return s.upsertContext(ctx, a, in, true)
}

func (s *Service) PrescriptionCatalogue(ctx context.Context, a Authorization) ([]CatalogueItem, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,category,code,display_name,active,display_order,version FROM development_prescription_catalogue ORDER BY category,display_order,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CatalogueItem{}
	for rows.Next() {
		var item CatalogueItem
		if err := rows.Scan(&item.ID, &item.Category, &item.Code, &item.DisplayName, &item.Active, &item.DisplayOrder, &item.Version); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Service) TheatreProcedureCatalogue(ctx context.Context, a Authorization) ([]CatalogueItem, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,code,display_name,active,display_order,version FROM development_theatre_procedure_catalogue ORDER BY display_order,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CatalogueItem{}
	for rows.Next() {
		var item CatalogueItem
		item.Category = "procedure"
		if err := rows.Scan(&item.ID, &item.Code, &item.DisplayName, &item.Active, &item.DisplayOrder, &item.Version); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Service) UpsertTheatreProcedureCatalogue(ctx context.Context, a Authorization, in CatalogueUpsert) (CatalogueItem, error) {
	if !validTheatreProcedure(in) {
		return CatalogueItem{}, ErrInvalidRequest
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return CatalogueItem{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var out CatalogueItem
	command := "theatre_procedure_catalogue.create"
	if in.ID == 0 {
		err = tx.QueryRow(ctx, `INSERT INTO development_theatre_procedure_catalogue(code,display_name,active,display_order) VALUES($1,$2,$3,$4) RETURNING id,code,display_name,active,display_order,version`, in.Code, in.DisplayName, in.Active, in.DisplayOrder).Scan(&out.ID, &out.Code, &out.DisplayName, &out.Active, &out.DisplayOrder, &out.Version)
	} else {
		command = "theatre_procedure_catalogue.update"
		err = tx.QueryRow(ctx, `UPDATE development_theatre_procedure_catalogue SET code=$1,display_name=$2,active=$3,display_order=$4,version=version+1 WHERE id=$5 AND version=$6 RETURNING id,code,display_name,active,display_order,version`, in.Code, in.DisplayName, in.Active, in.DisplayOrder, in.ID, in.ExpectedVersion).Scan(&out.ID, &out.Code, &out.DisplayName, &out.Active, &out.DisplayOrder, &out.Version)
		if errors.Is(err, pgx.ErrNoRows) {
			return CatalogueItem{}, ErrConflict
		}
	}
	if err != nil {
		return CatalogueItem{}, err
	}
	out.Category = "procedure"
	if err := auditContextTx(ctx, tx, a, command, "theatre_procedure_catalogue", out.ID, `["code","display_name","active","display_order"]`); err != nil {
		return CatalogueItem{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return CatalogueItem{}, err
	}
	return out, nil
}

func validTheatreProcedure(in CatalogueUpsert) bool {
	return !(in.ID > 0 && in.ExpectedVersion < 1) && in.DisplayOrder >= 0 && len(in.DisplayName) > 0 && len(in.DisplayName) <= 200 && len(in.Code) > 0 && len(in.Code) <= 120 && strings.HasPrefix(in.Code, "development_") && in.Category == "procedure"
}

func (s *Service) UpsertPrescriptionCatalogue(ctx context.Context, a Authorization, in CatalogueUpsert) (CatalogueItem, error) {
	if !validCatalogueItem(in) {
		return CatalogueItem{}, ErrInvalidRequest
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return CatalogueItem{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var out CatalogueItem
	command := "prescription_catalogue.create"
	if in.ID == 0 {
		err = tx.QueryRow(ctx, `INSERT INTO development_prescription_catalogue(category,code,display_name,active,display_order) VALUES($1,$2,$3,$4,$5) RETURNING id,category,code,display_name,active,display_order,version`, in.Category, in.Code, in.DisplayName, in.Active, in.DisplayOrder).Scan(&out.ID, &out.Category, &out.Code, &out.DisplayName, &out.Active, &out.DisplayOrder, &out.Version)
	} else {
		command = "prescription_catalogue.update"
		err = tx.QueryRow(ctx, `UPDATE development_prescription_catalogue SET category=$1,code=$2,display_name=$3,active=$4,display_order=$5,version=version+1 WHERE id=$6 AND version=$7 RETURNING id,category,code,display_name,active,display_order,version`, in.Category, in.Code, in.DisplayName, in.Active, in.DisplayOrder, in.ID, in.ExpectedVersion).Scan(&out.ID, &out.Category, &out.Code, &out.DisplayName, &out.Active, &out.DisplayOrder, &out.Version)
		if errors.Is(err, pgx.ErrNoRows) {
			return CatalogueItem{}, ErrConflict
		}
	}
	if err != nil {
		return CatalogueItem{}, err
	}
	if err := auditContextTx(ctx, tx, a, command, "prescription_catalogue", out.ID, `["category","code","display_name","active","display_order"]`); err != nil {
		return CatalogueItem{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return CatalogueItem{}, err
	}
	return out, nil
}

func validCatalogueItem(in CatalogueUpsert) bool {
	if in.ID > 0 && in.ExpectedVersion < 1 || in.DisplayOrder < 0 || len(in.DisplayName) == 0 || len(in.DisplayName) > 200 || len(in.Code) == 0 || len(in.Code) > 120 || !strings.HasPrefix(in.Code, "development_") {
		return false
	}
	switch in.Category {
	case "medication", "route", "frequency", "duration", "laterality":
		return true
	default:
		return false
	}
}

func (s *Service) upsertContext(ctx context.Context, a Authorization, in ContextUpsert, firm bool) (Reference, error) {
	name := strings.TrimSpace(in.Name)
	if len(name) > 120 || (in.ID > 0 && in.ExpectedVersion < 1) {
		return Reference{}, ErrInvalidRequest
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Reference{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var out Reference
	command := "site.create"
	targetType := "site"
	if firm {
		command, targetType = "firm.create", "firm"
	}
	if in.ID == 0 {
		if name == "" {
			return Reference{}, ErrInvalidRequest
		}
		if firm {
			err = tx.QueryRow(ctx, `INSERT INTO firms(institution_id,name,active) VALUES($1,$2,$3) RETURNING id,name,active,version`, a.principal.InstitutionID, name, in.Active).Scan(&out.ID, &out.Name, &out.Active, &out.Version)
		} else {
			err = tx.QueryRow(ctx, `INSERT INTO sites(institution_id,name,active) VALUES($1,$2,$3) RETURNING id,name,active,version`, a.principal.InstitutionID, name, in.Active).Scan(&out.ID, &out.Name, &out.Active, &out.Version)
		}
	} else {
		command = strings.TrimSuffix(command, ".create") + ".update"
		if name == "" {
			table := "sites"
			if firm {
				table = "firms"
			}
			if err = tx.QueryRow(ctx, fmt.Sprintf("SELECT name FROM %s WHERE id=$1 AND institution_id=$2", table), in.ID, a.principal.InstitutionID).Scan(&name); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return Reference{}, ErrConflict
				}
				return Reference{}, err
			}
		}
		if firm {
			err = tx.QueryRow(ctx, `UPDATE firms SET name=$1,active=$2,version=version+1,updated_at=now() WHERE id=$3 AND institution_id=$4 AND version=$5 RETURNING id,name,active,version`, name, in.Active, in.ID, a.principal.InstitutionID, in.ExpectedVersion).Scan(&out.ID, &out.Name, &out.Active, &out.Version)
		} else {
			err = tx.QueryRow(ctx, `UPDATE sites SET name=$1,active=$2,version=version+1,updated_at=now() WHERE id=$3 AND institution_id=$4 AND version=$5 RETURNING id,name,active,version`, name, in.Active, in.ID, a.principal.InstitutionID, in.ExpectedVersion).Scan(&out.ID, &out.Name, &out.Active, &out.Version)
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return Reference{}, ErrConflict
		}
	}
	if err != nil {
		return Reference{}, err
	}
	if err := auditContextTx(ctx, tx, a, command, targetType, out.ID, `["name","active"]`); err != nil {
		return Reference{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Reference{}, err
	}
	return out, nil
}

func auditContextTx(ctx context.Context, tx pgx.Tx, a Authorization, command, targetType string, targetID int64, fields string) error {
	_, err := tx.Exec(ctx, `INSERT INTO development_admin_audit(actor_user_id,institution_id,command,target_type,target_key,changed_fields,outcome,correlation_id) VALUES($1,$2,$3,$4,$5,$6::jsonb,'success',$7)`, a.principal.UserID, a.principal.InstitutionID, command, targetType, fmt.Sprintf("%s:%d", targetType, targetID), fields, a.metadata.CorrelationID)
	return err
}
func validSettingKey(k string) bool {
	switch k {
	case "default_site", "default_firm", "appointment_slot_minutes", "demo_retention_days",
		"clinic_flow_queue_name", "clinic_flow_priorities", "theatre_default_room",
		"theatre_session_minutes", "theatre_capacity_minutes", "referral_default_recipient",
		"referral_default_priority", "correspondence_default_template", "correspondence_footer",
		"messaging_default_type", "messaging_mailbox":
		return true
	}
	return false
}

func validSettingValue(key, value string) bool {
	value = strings.TrimSpace(value)
	if key == "" || value == "" || len(value) > 80 || !validSettingKey(key) {
		return false
	}
	switch key {
	case "appointment_slot_minutes", "demo_retention_days", "theatre_session_minutes", "theatre_capacity_minutes":
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 || n > 1440 {
			return false
		}
	case "clinic_flow_priorities":
		for _, priority := range strings.Split(value, ",") {
			priority = strings.TrimSpace(priority)
			if priority == "" || (priority != "routine" && priority != "urgent") {
				return false
			}
		}
	case "default_site", "default_firm":
		return true
	default:
		if strings.Contains(value, "\n") || strings.Contains(value, "\r") {
			return false
		}
	}
	return true
}

var _ = pgx.ErrNoRows
