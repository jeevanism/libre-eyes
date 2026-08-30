package referralappointment

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jeevanism/libre-eyes/internal/auth"
	"strings"
	"time"
)

type operationAuthorizer interface {
	AuthorizeOperation(context.Context, auth.OperationAuthorizationRequest) (auth.OperationPrincipal, error)
}
type Service struct {
	pool       *pgxpool.Pool
	authorizer operationAuthorizer
}

func NewService(pool *pgxpool.Pool, authorizer operationAuthorizer) (*Service, error) {
	if pool == nil || authorizer == nil {
		return nil, errors.New("referral appointment database and authorizer are required")
	}
	return &Service{pool: pool, authorizer: authorizer}, nil
}
func (s *Service) Authorize(ctx context.Context, token, csrf string, metadata auth.RequestMetadata) (Authorization, error) {
	p, e := s.authorizer.AuthorizeOperation(ctx, auth.OperationAuthorizationRequest{Token: token, CSRFToken: csrf, Permission: PermissionManage, Capability: "referrals", DeniedEventType: "referral.development_appointment.denied", Metadata: metadata})
	if e != nil {
		return Authorization{}, e
	}
	return Authorization{principalUserID: p.UserID, institutionID: p.InstitutionID, siteID: p.SiteID, firmID: p.FirmID, correlationID: metadata.CorrelationID, service: s}, nil
}
func (s *Service) valid(a Authorization) bool { return a.service == s && a.principalUserID > 0 }
func (s *Service) List(ctx context.Context, a Authorization) ([]Request, error) {
	if !s.valid(a) {
		return nil, ErrInvalidRequest
	}
	rows, e := s.pool.Query(ctx, `SELECT id::text,synthetic_patient_label,recipient_role,clinic_code,appointment_date::text,COALESCE(to_char(appointment_time,'HH24:MI'),''),priority,notes,status,version FROM development_referral_appointments WHERE institution_id=$1 AND site_id=$2 AND firm_id=$3 AND owner_user_id=$4 AND expires_at>now() ORDER BY appointment_date,appointment_time NULLS LAST,id`, a.institutionID, a.siteID, a.firmID, a.principalUserID)
	if e != nil {
		return nil, fmt.Errorf("list development referrals: %w", e)
	}
	defer rows.Close()
	out := []Request{}
	for rows.Next() {
		var r Request
		if e := rows.Scan(&r.ID, &r.SyntheticPatientLabel, &r.RecipientRole, &r.ClinicCode, &r.AppointmentDate, &r.AppointmentTime, &r.Priority, &r.Notes, &r.Status, &r.Version); e != nil {
			return nil, e
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
func (s *Service) Create(ctx context.Context, a Authorization, in CreateRequest) (Request, error) {
	if !s.valid(a) || !validCreate(in) {
		return Request{}, ErrInvalidRequest
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Request{}, fmt.Errorf("begin development referral create: %w", err)
	}
	defer tx.Rollback(ctx)
	id := fmt.Sprintf("%s", newUUID())
	retention := in.RetentionKind
	if retention == "" {
		retention = "manual"
	}
	expires := time.Now().UTC().Add(7 * 24 * time.Hour)
	if retention == "autosave" {
		expires = time.Now().UTC().Add(24 * time.Hour)
	}
	var r Request
	e := tx.QueryRow(ctx, `INSERT INTO development_referral_appointments(id,institution_id,site_id,firm_id,synthetic_patient_id,synthetic_patient_label,recipient_role,clinic_code,appointment_date,appointment_time,priority,notes,owner_user_id,expires_at,retention_kind) VALUES($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9::date,NULLIF($10,'')::time,$11,$12,$13,$14,$15) RETURNING id::text,synthetic_patient_label,recipient_role,clinic_code,appointment_date::text,COALESCE(to_char(appointment_time,'HH24:MI'),''),priority,notes,status,version`, id, a.institutionID, a.siteID, a.firmID, in.SyntheticPatientID, in.SyntheticPatientLabel, in.RecipientRole, in.ClinicCode, in.AppointmentDate, in.AppointmentTime, in.Priority, in.Notes, a.principalUserID, expires, retention).Scan(&r.ID, &r.SyntheticPatientLabel, &r.RecipientRole, &r.ClinicCode, &r.AppointmentDate, &r.AppointmentTime, &r.Priority, &r.Notes, &r.Status, &r.Version)
	if e != nil {
		return Request{}, fmt.Errorf("create development referral: %w", e)
	}
	if _, e = tx.Exec(ctx, `INSERT INTO development_referral_audit(referral_id,actor_user_id,institution_id,site_id,firm_id,command,next_state,resulting_version,correlation_id) VALUES($1::uuid,$2,$3,$4,$5,'create',$6::development_referral_status,$7,$8)`, r.ID, a.principalUserID, a.institutionID, a.siteID, a.firmID, r.Status, r.Version, a.correlationID); e != nil {
		return Request{}, fmt.Errorf("audit development referral create: %w", e)
	}
	if e = tx.Commit(ctx); e != nil {
		return Request{}, fmt.Errorf("commit development referral create: %w", e)
	}
	return r, nil
}
func (s *Service) Command(ctx context.Context, a Authorization, in CommandRequest) (Request, error) {
	if !s.valid(a) || !validUUID(in.RequestID) || in.ExpectedVersion < 1 {
		return Request{}, ErrInvalidRequest
	}
	tx, e := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if e != nil {
		return Request{}, e
	}
	defer tx.Rollback(ctx)
	var cur Request
	var owner int64
	e = tx.QueryRow(ctx, `SELECT id::text,synthetic_patient_label,recipient_role,clinic_code,appointment_date::text,COALESCE(to_char(appointment_time,'HH24:MI'),''),priority,notes,status,version,owner_user_id FROM development_referral_appointments WHERE id=$1::uuid AND institution_id=$2 AND site_id=$3 AND firm_id=$4 AND owner_user_id=$5 FOR UPDATE`, in.RequestID, a.institutionID, a.siteID, a.firmID, a.principalUserID).Scan(&cur.ID, &cur.SyntheticPatientLabel, &cur.RecipientRole, &cur.ClinicCode, &cur.AppointmentDate, &cur.AppointmentTime, &cur.Priority, &cur.Notes, &cur.Status, &cur.Version, &owner)
	if errors.Is(e, pgx.ErrNoRows) {
		return Request{}, ErrNotFound
	}
	if e != nil {
		return Request{}, e
	}
	if cur.Version != in.ExpectedVersion {
		return Request{}, ErrConflict
	}
	next, e := transition(cur.Status, in.Command)
	if e != nil {
		return Request{}, e
	}
	var out Request
	e = tx.QueryRow(ctx, `UPDATE development_referral_appointments SET status=$1::development_referral_status,version=version+1,updated_at=now() WHERE id=$2::uuid AND version=$3 RETURNING id::text,synthetic_patient_label,recipient_role,clinic_code,appointment_date::text,COALESCE(to_char(appointment_time,'HH24:MI'),''),priority,notes,status,version`, next, cur.ID, cur.Version).Scan(&out.ID, &out.SyntheticPatientLabel, &out.RecipientRole, &out.ClinicCode, &out.AppointmentDate, &out.AppointmentTime, &out.Priority, &out.Notes, &out.Status, &out.Version)
	if errors.Is(e, pgx.ErrNoRows) {
		return Request{}, ErrConflict
	}
	if e != nil {
		return Request{}, e
	}
	if _, e = tx.Exec(ctx, `INSERT INTO development_referral_audit(referral_id,actor_user_id,institution_id,site_id,firm_id,command,prior_state,next_state,resulting_version,correlation_id) VALUES($1::uuid,$2,$3,$4,$5,$6,$7::development_referral_status,$8::development_referral_status,$9,$10)`, out.ID, a.principalUserID, a.institutionID, a.siteID, a.firmID, in.Command, cur.Status, out.Status, out.Version, a.correlationID); e != nil {
		return Request{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return Request{}, e
	}
	return out, nil
}
func transition(cur Status, c Command) (Status, error) {
	switch c {
	case CommandSchedule:
		if cur != StatusRequested {
			return "", ErrInvalidTransition
		}
		return StatusScheduled, nil
	case CommandArrive:
		if cur != StatusScheduled {
			return "", ErrInvalidTransition
		}
		return StatusArrived, nil
	case CommandComplete:
		if cur != StatusArrived {
			return "", ErrInvalidTransition
		}
		return StatusCompleted, nil
	case CommandAbandon:
		if cur == StatusCompleted || cur == StatusAbandoned {
			return "", ErrInvalidTransition
		}
		return StatusAbandoned, nil
	}
	return "", ErrInvalidRequest
}
func validCreate(r CreateRequest) bool {
	if !validUUID(r.SyntheticPatientID) || strings.TrimSpace(r.SyntheticPatientLabel) == "" || len(r.SyntheticPatientLabel) > 200 || len(r.Notes) > 1000 || strings.Contains(r.Notes, "<") {
		return false
	}
	if r.RecipientRole != "demo_gp" && r.RecipientRole != "demo_optometrist" && r.RecipientRole != "demo_consultant" {
		return false
	}
	if r.ClinicCode != "demo_general_eye_clinic" && r.ClinicCode != "demo_glaucoma_clinic" && r.ClinicCode != "demo_retina_clinic" {
		return false
	}
	if r.Priority != "routine" && r.Priority != "soon" && r.Priority != "urgent" {
		return false
	}
	parsedDate, e := time.Parse("2006-01-02", r.AppointmentDate)
	if e != nil || parsedDate.Format("2006-01-02") != r.AppointmentDate || parsedDate.Before(time.Now().UTC().Truncate(24*time.Hour)) {
		return false
	}
	if r.AppointmentTime != "" {
		if parsedTime, err := time.Parse("15:04", r.AppointmentTime); err != nil || parsedTime.Format("15:04") != r.AppointmentTime {
			return false
		}
	}
	return r.RetentionKind == "" || r.RetentionKind == "autosave" || r.RetentionKind == "manual"
}
func validUUID(v string) bool {
	if len(v) != 36 {
		return false
	}
	for i, c := range v {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
			continue
		}
		if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
			return false
		}
	}
	return true
}
func newUUID() string {
	b := make([]byte, 16)
	if _, e := rand.Read(b); e != nil {
		return "77777777-7777-4777-8777-777777777777"
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	encoded := hex.EncodeToString(b)
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32]
}
