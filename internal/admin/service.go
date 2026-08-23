package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

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
		out = append(out, u)
	}
	return out, rows.Err()
}
func (s *Service) Contexts(ctx context.Context, a Authorization) (Contexts, error) {
	var out Contexts
	if err := s.pool.QueryRow(ctx, `SELECT id,name FROM institutions WHERE id=$1`, a.principal.InstitutionID).Scan(&out.Institution.ID, &out.Institution.Name); err != nil {
		return out, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id,name FROM sites WHERE institution_id=$1 AND active ORDER BY name`, a.principal.InstitutionID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var r Reference
		if err := rows.Scan(&r.ID, &r.Name); err != nil {
			return out, err
		}
		out.Sites = append(out.Sites, r)
	}
	rows, err = s.pool.Query(ctx, `SELECT id,name FROM firms WHERE institution_id=$1 AND active ORDER BY name`, a.principal.InstitutionID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var r Reference
		if err := rows.Scan(&r.ID, &r.Name); err != nil {
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
	rows, err := s.pool.Query(ctx, `SELECT command,target_type,changed_fields,outcome FROM development_admin_audit WHERE institution_id=$1 ORDER BY created_at DESC,id DESC LIMIT 100`, a.principal.InstitutionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AuditEvent{}
	for rows.Next() {
		var v AuditEvent
		var raw []byte
		if err := rows.Scan(&v.Command, &v.TargetType, &raw, &v.Outcome); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(raw, &v.ChangedFields)
		out = append(out, v)
	}
	return out, rows.Err()
}

var _ = pgx.ErrNoRows
