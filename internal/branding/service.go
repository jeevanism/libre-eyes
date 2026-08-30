package branding

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jeevanism/libre-eyes/internal/auth"
)

type authorizer interface {
	AuthorizeOperation(context.Context, auth.OperationAuthorizationRequest) (auth.OperationPrincipal, error)
	AuthorizeRead(context.Context, auth.ReadAuthorizationRequest) (auth.OperationPrincipal, error)
}

// Authorization is a server-derived branding administration scope.
type Authorization struct {
	principal auth.OperationPrincipal
	metadata  auth.RequestMetadata
	service   *Service
}

// Service owns presentation profile reads and versioned administration commands.
type Service struct {
	pool       *pgxpool.Pool
	authorizer authorizer
}

// NewService constructs a branding service with explicit dependencies.
func NewService(pool *pgxpool.Pool, a authorizer) (*Service, error) {
	if pool == nil || a == nil {
		return nil, errors.New("branding database and authorizer are required")
	}
	return &Service{pool: pool, authorizer: a}, nil
}

// Authorize derives the institution scope from the authenticated session.
func (s *Service) Authorize(ctx context.Context, token, csrf string, metadata auth.RequestMetadata, write bool) (Authorization, error) {
	var principal auth.OperationPrincipal
	var err error
	if write {
		principal, err = s.authorizer.AuthorizeOperation(ctx, auth.OperationAuthorizationRequest{
			Token: token, CSRFToken: csrf, Permission: permissionManage,
			DeniedEventType: "branding.denied", Metadata: metadata,
		})
	} else {
		principal, err = s.authorizer.AuthorizeRead(ctx, auth.ReadAuthorizationRequest{
			Token: token, Permission: permissionRead, DeniedEventType: "branding.denied", Metadata: metadata,
		})
	}
	if err != nil {
		return Authorization{}, err
	}

	var role string
	err = s.pool.QueryRow(ctx, `
		SELECT role_code
		FROM development_admin_users
		WHERE user_id=$1 AND institution_id=$2 AND active
		  AND role_code IN ('system_administrator','institution_administrator')
		LIMIT 1`, principal.UserID, principal.InstitutionID).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Authorization{}, ErrForbidden
		}
		return Authorization{}, fmt.Errorf("check branding administration role: %w", err)
	}
	return Authorization{principal: principal, metadata: metadata, service: s}, nil
}

// PublicProfile returns only a published profile or the embedded fallback.
func (s *Service) PublicProfile(ctx context.Context, institutionID int64) (Profile, error) {
	profile := DefaultProfile()
	query := `
		SELECT v.institution_id,v.profile_version,v.row_version,v.status,
		       v.organization_name,v.short_name,v.browser_title,
		       v.primary_color,v.primary_hover_color,v.selected_surface_color,v.focus_color
		FROM development_branding_profile_versions v
		JOIN institutions i ON i.id=v.institution_id AND i.active
		WHERE v.status='published'`
	args := []any{}
	if institutionID > 0 {
		query += ` AND i.id=$1`
		args = append(args, institutionID)
	}
	query += ` ORDER BY i.id LIMIT 1`
	loaded, err := scanProfile(s.pool.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return profile, nil
	}
	if err != nil {
		return Profile{}, fmt.Errorf("load published branding profile: %w", err)
	}
	loaded.Source = "published"
	return loaded, nil
}

// State returns the current institution's effective, draft, history, and audit views.
func (s *Service) State(ctx context.Context, authorization Authorization) (State, error) {
	if authorization.service != s {
		return State{}, ErrForbidden
	}
	return s.loadState(ctx, authorization.principal.InstitutionID)
}

// SaveDraft creates or updates the single institution branding draft.
func (s *Service) SaveDraft(ctx context.Context, authorization Authorization, input DraftInput) (State, error) {
	if authorization.service != s {
		return State{}, ErrForbidden
	}
	input = normalizeInput(input)
	if err := validateInput(input); err != nil || input.ExpectedVersion < 0 {
		return State{}, ErrInvalidRequest
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return State{}, fmt.Errorf("begin branding draft transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	institutionID := authorization.principal.InstitutionID
	current, err := scanProfile(tx.QueryRow(ctx, profileSelect+` WHERE institution_id=$1 AND status='draft' FOR UPDATE`, institutionID))
	var saved Profile
	var before map[string]any
	if errors.Is(err, pgx.ErrNoRows) {
		if input.ExpectedVersion != 0 {
			return State{}, ErrConflict
		}
		if err := tx.QueryRow(ctx, `SELECT id FROM institutions WHERE id=$1 AND active FOR UPDATE`, institutionID).Scan(&institutionID); err != nil {
			return State{}, fmt.Errorf("lock branding institution: %w", err)
		}
		var profileVersion int64
		if err := tx.QueryRow(ctx, `SELECT COALESCE(max(profile_version),0)+1 FROM development_branding_profile_versions WHERE institution_id=$1`, institutionID).Scan(&profileVersion); err != nil {
			return State{}, fmt.Errorf("allocate branding profile version: %w", err)
		}
		err = tx.QueryRow(ctx, `
			INSERT INTO development_branding_profile_versions (
				institution_id,profile_version,status,organization_name,short_name,browser_title,
				primary_color,primary_hover_color,selected_surface_color,focus_color,created_by_user_id
			) VALUES ($1,$2,'draft',$3,$4,$5,$6,$7,$8,$9,$10)
			RETURNING institution_id,profile_version,row_version,status,organization_name,short_name,browser_title,
			          primary_color,primary_hover_color,selected_surface_color,focus_color`,
			institutionID, profileVersion, input.OrganizationName, input.ShortName, input.BrowserTitle,
			input.Colors.Primary, input.Colors.PrimaryHover, input.Colors.SelectedSurface, input.Colors.Focus,
			authorization.principal.UserID).Scan(profileDestinations(&saved)...)
		before = map[string]any{}
	} else if err != nil {
		return State{}, fmt.Errorf("load branding draft: %w", err)
	} else {
		if current.RowVersion != input.ExpectedVersion {
			return State{}, ErrConflict
		}
		before = profileSnapshot(current)
		err = tx.QueryRow(ctx, `
			UPDATE development_branding_profile_versions
			SET organization_name=$1,short_name=$2,browser_title=$3,primary_color=$4,
			    primary_hover_color=$5,selected_surface_color=$6,focus_color=$7,
			    row_version=row_version+1,updated_at=now()
			WHERE institution_id=$8 AND status='draft' AND row_version=$9
			RETURNING institution_id,profile_version,row_version,status,organization_name,short_name,browser_title,
			          primary_color,primary_hover_color,selected_surface_color,focus_color`,
			input.OrganizationName, input.ShortName, input.BrowserTitle, input.Colors.Primary,
			input.Colors.PrimaryHover, input.Colors.SelectedSurface, input.Colors.Focus,
			institutionID, input.ExpectedVersion).Scan(profileDestinations(&saved)...)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return State{}, ErrConflict
	}
	if err != nil {
		return State{}, fmt.Errorf("save branding draft: %w", err)
	}
	if err := insertAudit(ctx, tx, authorization, "branding.draft_saved", saved.ProfileVersion, before, profileSnapshot(saved)); err != nil {
		return State{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return State{}, fmt.Errorf("commit branding draft: %w", err)
	}
	return s.loadState(ctx, institutionID)
}

// Publish promotes the current draft while retaining the previous published version.
func (s *Service) Publish(ctx context.Context, authorization Authorization, command VersionCommand) (State, error) {
	if authorization.service != s {
		return State{}, ErrForbidden
	}
	if command.ExpectedVersion < 1 {
		return State{}, ErrInvalidRequest
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return State{}, fmt.Errorf("begin branding publish transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	institutionID := authorization.principal.InstitutionID
	draft, err := scanProfile(tx.QueryRow(ctx, profileSelect+` WHERE institution_id=$1 AND status='draft' FOR UPDATE`, institutionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return State{}, ErrNoDraft
	}
	if err != nil {
		return State{}, fmt.Errorf("load branding draft for publication: %w", err)
	}
	if draft.RowVersion != command.ExpectedVersion {
		return State{}, ErrConflict
	}
	current, err := scanProfile(tx.QueryRow(ctx, profileSelect+` WHERE institution_id=$1 AND status='published' FOR UPDATE`, institutionID))
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return State{}, fmt.Errorf("load current published branding: %w", err)
	}
	before := map[string]any{}
	if err == nil {
		before = profileSnapshot(current)
		if _, err := tx.Exec(ctx, `UPDATE development_branding_profile_versions SET status='superseded',row_version=row_version+1,updated_at=now() WHERE institution_id=$1 AND status='published'`, institutionID); err != nil {
			return State{}, fmt.Errorf("supersede published branding: %w", err)
		}
	}
	var published Profile
	err = tx.QueryRow(ctx, `
		UPDATE development_branding_profile_versions
		SET status='published',row_version=row_version+1,published_by_user_id=$1,published_at=now(),updated_at=now()
		WHERE institution_id=$2 AND status='draft' AND row_version=$3
		RETURNING institution_id,profile_version,row_version,status,organization_name,short_name,browser_title,
		          primary_color,primary_hover_color,selected_surface_color,focus_color`,
		authorization.principal.UserID, institutionID, command.ExpectedVersion).Scan(profileDestinations(&published)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return State{}, ErrConflict
	}
	if err != nil {
		return State{}, fmt.Errorf("publish branding draft: %w", err)
	}
	if err := insertAudit(ctx, tx, authorization, "branding.published", published.ProfileVersion, before, profileSnapshot(published)); err != nil {
		return State{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return State{}, fmt.Errorf("commit branding publication: %w", err)
	}
	return s.loadState(ctx, institutionID)
}

// Rollback republishes a superseded profile as a new version, preserving history.
func (s *Service) Rollback(ctx context.Context, authorization Authorization, command VersionCommand) (State, error) {
	if authorization.service != s {
		return State{}, ErrForbidden
	}
	if command.ExpectedVersion < 1 || command.TargetProfileVersion < 1 {
		return State{}, ErrInvalidRequest
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return State{}, fmt.Errorf("begin branding rollback transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	institutionID := authorization.principal.InstitutionID
	current, err := scanProfile(tx.QueryRow(ctx, profileSelect+` WHERE institution_id=$1 AND status='published' FOR UPDATE`, institutionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return State{}, ErrNotFound
	}
	if err != nil {
		return State{}, fmt.Errorf("load published branding for rollback: %w", err)
	}
	if current.RowVersion != command.ExpectedVersion {
		return State{}, ErrConflict
	}
	target, err := scanProfile(tx.QueryRow(ctx, profileSelect+` WHERE institution_id=$1 AND profile_version=$2 AND status='superseded' FOR UPDATE`, institutionID, command.TargetProfileVersion))
	if errors.Is(err, pgx.ErrNoRows) {
		return State{}, ErrNotFound
	}
	if err != nil {
		return State{}, fmt.Errorf("load branding rollback target: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE development_branding_profile_versions SET status='superseded',row_version=row_version+1,updated_at=now() WHERE institution_id=$1 AND status='published' AND row_version=$2`, institutionID, command.ExpectedVersion); err != nil {
		return State{}, fmt.Errorf("supersede branding before rollback: %w", err)
	}
	var nextVersion int64
	if err := tx.QueryRow(ctx, `SELECT max(profile_version)+1 FROM development_branding_profile_versions WHERE institution_id=$1`, institutionID).Scan(&nextVersion); err != nil {
		return State{}, fmt.Errorf("allocate rollback profile version: %w", err)
	}
	var published Profile
	err = tx.QueryRow(ctx, `
		INSERT INTO development_branding_profile_versions (
			institution_id,profile_version,status,organization_name,short_name,browser_title,
			primary_color,primary_hover_color,selected_surface_color,focus_color,
			created_by_user_id,published_by_user_id,published_at
		) VALUES ($1,$2,'published',$3,$4,$5,$6,$7,$8,$9,$10,$10,now())
		RETURNING institution_id,profile_version,row_version,status,organization_name,short_name,browser_title,
		          primary_color,primary_hover_color,selected_surface_color,focus_color`,
		institutionID, nextVersion, target.OrganizationName, target.ShortName, target.BrowserTitle,
		target.Colors.Primary, target.Colors.PrimaryHover, target.Colors.SelectedSurface, target.Colors.Focus,
		authorization.principal.UserID).Scan(profileDestinations(&published)...)
	if err != nil {
		return State{}, fmt.Errorf("publish branding rollback: %w", err)
	}
	if err := insertAudit(ctx, tx, authorization, "branding.rolled_back", published.ProfileVersion, profileSnapshot(current), profileSnapshot(published)); err != nil {
		return State{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return State{}, fmt.Errorf("commit branding rollback: %w", err)
	}
	return s.loadState(ctx, institutionID)
}

const profileSelect = `
	SELECT institution_id,profile_version,row_version,status,organization_name,short_name,browser_title,
	       primary_color,primary_hover_color,selected_surface_color,focus_color
	FROM development_branding_profile_versions`

type scanner interface {
	Scan(...any) error
}

func scanProfile(row scanner) (Profile, error) {
	var profile Profile
	err := row.Scan(profileDestinations(&profile)...)
	if err != nil {
		return Profile{}, err
	}
	profile.Source = profile.Status
	return profile, nil
}

func profileDestinations(profile *Profile) []any {
	return []any{
		&profile.InstitutionID, &profile.ProfileVersion, &profile.RowVersion, &profile.Status,
		&profile.OrganizationName, &profile.ShortName, &profile.BrowserTitle,
		&profile.Colors.Primary, &profile.Colors.PrimaryHover, &profile.Colors.SelectedSurface, &profile.Colors.Focus,
	}
}

func profileSnapshot(profile Profile) map[string]any {
	return map[string]any{
		"profileVersion":   profile.ProfileVersion,
		"organizationName": profile.OrganizationName,
		"shortName":        profile.ShortName,
		"browserTitle":     profile.BrowserTitle,
		"colors":           profile.Colors,
	}
}

func insertAudit(ctx context.Context, tx pgx.Tx, authorization Authorization, command string, profileVersion int64, before, after map[string]any) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO development_branding_audit (
			actor_user_id,institution_id,command,profile_version,before_values,after_values,correlation_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		authorization.principal.UserID, authorization.principal.InstitutionID, command, profileVersion,
		before, after, authorization.metadata.CorrelationID)
	if err != nil {
		return fmt.Errorf("append branding audit event: %w", err)
	}
	return nil
}

func (s *Service) loadState(ctx context.Context, institutionID int64) (State, error) {
	state := State{History: []Summary{}, Audit: []AuditEvent{}}
	published, err := scanProfile(s.pool.QueryRow(ctx, profileSelect+` WHERE institution_id=$1 AND status='published'`, institutionID))
	if err == nil {
		published.Source = "published"
		state.Published = &published
		state.Effective = published
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return State{}, fmt.Errorf("load branding published state: %w", err)
	}
	if state.Published == nil {
		state.Effective = DefaultProfile()
		state.Effective.InstitutionID = institutionID
	}
	draft, err := scanProfile(s.pool.QueryRow(ctx, profileSelect+` WHERE institution_id=$1 AND status='draft'`, institutionID))
	if err == nil {
		draft.Source = "draft"
		state.Draft = &draft
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return State{}, fmt.Errorf("load branding draft state: %w", err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT profile_version,row_version,status,updated_at,published_at
		FROM development_branding_profile_versions
		WHERE institution_id=$1 ORDER BY profile_version DESC LIMIT 20`, institutionID)
	if err != nil {
		return State{}, fmt.Errorf("list branding history: %w", err)
	}
	for rows.Next() {
		var summary Summary
		if err := rows.Scan(&summary.ProfileVersion, &summary.RowVersion, &summary.Status, &summary.UpdatedAt, &summary.PublishedAt); err != nil {
			rows.Close()
			return State{}, fmt.Errorf("scan branding history: %w", err)
		}
		state.History = append(state.History, summary)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return State{}, fmt.Errorf("iterate branding history: %w", err)
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `
		SELECT a.command,a.profile_version,u.display_name,a.before_values,a.after_values,a.correlation_id,a.created_at
		FROM development_branding_audit a
		JOIN users u ON u.id=a.actor_user_id
		WHERE a.institution_id=$1 ORDER BY a.created_at DESC,a.id DESC LIMIT 20`, institutionID)
	if err != nil {
		return State{}, fmt.Errorf("list branding audit: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var event AuditEvent
		if err := rows.Scan(&event.Command, &event.ProfileVersion, &event.ActorDisplayName, &event.Before, &event.After, &event.CorrelationID, &event.CreatedAt); err != nil {
			return State{}, fmt.Errorf("scan branding audit: %w", err)
		}
		state.Audit = append(state.Audit, event)
	}
	if err := rows.Err(); err != nil {
		return State{}, fmt.Errorf("iterate branding audit: %w", err)
	}
	return state, nil
}
