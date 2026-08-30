//go:build integration

package migrations

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestEpisodesEventsFoundationConstraints(t *testing.T) {
	pool := patientSearchTestPool(t)

	t.Run("approved permissions exist without role assignment", func(t *testing.T) {
		var count int
		if err := pool.QueryRow(context.Background(), `
			SELECT count(*)
			FROM permissions
			WHERE name = ANY($1::text[])`, []string{
			"episode.read",
			"episode.read.override",
			"episode.create",
			"episode.update",
			"episode.reopen",
			"event_draft.create",
			"event_draft.read",
			"event_draft.update",
			"event_draft.abandon",
		}).Scan(&count); err != nil {
			t.Fatalf("count episode permissions: %v", err)
		}
		if count != 9 {
			t.Fatalf("episode permission count = %d, want 9", count)
		}

		var assignments int
		if err := pool.QueryRow(context.Background(), `
			SELECT count(*)
			FROM user_role_assignments ura
			JOIN role_permissions rp ON rp.role_id = ura.role_id AND rp.active
			JOIN permissions p ON p.id = rp.permission_id
			WHERE ura.active AND (
				p.name LIKE 'episode.%' OR p.name LIKE 'event_draft.%'
			)`,
		).Scan(&assignments); err != nil {
			t.Fatalf("count episode permission assignments: %v", err)
		}
		if assignments != 0 {
			t.Fatalf("episode permissions have %d existing role assignments, want none", assignments)
		}
	})

	t.Run("episode requires a matching patient institution association", func(t *testing.T) {
		withPatientSearchTx(t, pool, func(ctx context.Context, tx pgx.Tx) {
			fixture := insertEpisodeFixture(ctx, t, tx, "scope")
			otherInstitution := insertInstitution(ctx, t, tx, "Other Episode Institution")
			_, err := tx.Exec(ctx, `
				INSERT INTO episodes (
					public_id, patient_id, institution_id, patient_institution_id,
					site_id, firm_id, status, started_at
				) VALUES (
					'11111111-1111-4111-8111-111111111308', $1, $2, $3,
					$4, $5, 'open', now()
				)`, fixture.patientID, otherInstitution, fixture.patientInstitutionID, fixture.siteID, fixture.firmID)
			expectPostgresCode(t, err, "23503")
		})
	})

	t.Run("closed episode requires an end timestamp", func(t *testing.T) {
		withPatientSearchTx(t, pool, func(ctx context.Context, tx pgx.Tx) {
			fixture := insertEpisodeFixture(ctx, t, tx, "closed")
			_, err := tx.Exec(ctx, `
				INSERT INTO episodes (
					public_id, patient_id, institution_id, patient_institution_id,
					site_id, firm_id, status, started_at
				) VALUES (
					'11111111-1111-4111-8111-111111111302', $1, $2, $3,
					$4, $5, 'closed', now()
				)`, fixture.patientID, fixture.institutionID, fixture.patientInstitutionID, fixture.siteID, fixture.firmID)
			expectPostgresCode(t, err, "23514")
		})
	})

	t.Run("event versions are append only and tied to their episode", func(t *testing.T) {
		withPatientSearchTx(t, pool, func(ctx context.Context, tx pgx.Tx) {
			fixture := insertEpisodeFixture(ctx, t, tx, "event-version")
			eventID := insertEvent(ctx, t, tx, fixture, "11111111-1111-4111-8111-111111111303")
			if _, err := tx.Exec(ctx, `
				INSERT INTO event_versions (
					event_id, episode_id, episode_audit_sequence,
					resulting_version, command, actor_user_id, correlation_id
				) VALUES ($1, $2, 1, 1, 'import_baseline', $3, 'episode-event-test')`, eventID, fixture.episodeID, fixture.ownerUserID); err != nil {
				t.Fatalf("insert event version: %v", err)
			}
			expectConstraintInSavepoint(t, ctx, tx, "23514", `
				INSERT INTO event_versions (
					event_id, episode_id, episode_audit_sequence,
					resulting_version, command, correlation_id
				) VALUES ($1, $2, 2, 2, 'unattributed', 'episode-event-test')`, eventID, fixture.episodeID)
			expectConstraintInSavepoint(t, ctx, tx, "P0001", "UPDATE event_versions SET command = 'changed' WHERE event_id = $1", eventID)
			expectConstraintInSavepoint(t, ctx, tx, "P0001", "DELETE FROM event_versions WHERE event_id = $1", eventID)
			expectConstraintInSavepoint(t, ctx, tx, "P0001", "TRUNCATE event_versions")
		})
	})

	t.Run("orphan events retain a valid site and institution scope", func(t *testing.T) {
		withPatientSearchTx(t, pool, func(ctx context.Context, tx pgx.Tx) {
			fixture := insertEpisodeFixture(ctx, t, tx, "orphan-scope")
			otherInstitution := insertInstitution(ctx, t, tx, "Orphan Other Institution")
			var otherSiteID int64
			if err := tx.QueryRow(ctx, `
				INSERT INTO sites (institution_id, name)
				VALUES ($1, 'Orphan Other Site')
				RETURNING id`, otherInstitution).Scan(&otherSiteID); err != nil {
				t.Fatalf("insert other site: %v", err)
			}
			expectConstraintInSavepoint(t, ctx, tx, "23503", `
				INSERT INTO events (
					public_id, patient_id, institution_id, patient_institution_id, site_id, firm_id,
					event_type_code, occurred_at, is_imported_orphan
				) VALUES (
					'11111111-1111-4111-8111-111111111309', $1, $2, $3, $4, $5,
					'core.examination', now(), TRUE
				)`, fixture.patientID, fixture.institutionID, fixture.patientInstitutionID, otherSiteID, fixture.firmID)
		})
	})

	t.Run("deleted events require a meaningful deletion reason", func(t *testing.T) {
		withPatientSearchTx(t, pool, func(ctx context.Context, tx pgx.Tx) {
			fixture := insertEpisodeFixture(ctx, t, tx, "deletion-reason")
			eventID := insertEvent(ctx, t, tx, fixture, "11111111-1111-4111-8111-111111111310")
			expectConstraintInSavepoint(t, ctx, tx, "23514", `
				UPDATE events
				SET status = 'deleted', deleted_at = now(), deleted_by_user_id = $1, deletion_reason = '   '
				WHERE id = $2`, fixture.ownerUserID, eventID)
		})
	})

	t.Run("draft intent, payload, and active uniqueness are enforced", func(t *testing.T) {
		withPatientSearchTx(t, pool, func(ctx context.Context, tx pgx.Tx) {
			fixture := insertEpisodeFixture(ctx, t, tx, "draft")
			insertDraft(t, tx, fixture, "11111111-1111-4111-8111-111111111304", "create", "autosave", nil)
			expectConstraintInSavepoint(t, ctx, tx, "23505", `
				INSERT INTO event_drafts (
					public_id, owner_user_id, episode_id, patient_id, institution_id, site_id, firm_id,
					event_type_code, intent, mode, schema_version, payload, expires_at
				) VALUES (
					'11111111-1111-4111-8111-111111111305', $1, $2, $3, $4, $5, $6,
					'core.examination', 'create', 'autosave', 1, '{}'::jsonb, now() + interval '1 day'
				)`, fixture.ownerUserID, fixture.episodeID, fixture.patientID, fixture.institutionID, fixture.siteID, fixture.firmID)

			expectConstraintInSavepoint(t, ctx, tx, "23514", `
				INSERT INTO event_drafts (
					public_id, owner_user_id, episode_id, patient_id, institution_id, site_id, firm_id,
					event_type_code, intent, mode, schema_version, payload, expires_at
				) VALUES (
					'11111111-1111-4111-8111-111111111306', $1, $2, $3, $4, $5, $6,
					'core.examination', 'update', 'manual', 1, '{}'::jsonb, now() + interval '1 day'
				)`, fixture.ownerUserID, fixture.episodeID, fixture.patientID, fixture.institutionID, fixture.siteID, fixture.firmID)

			expectConstraintInSavepoint(t, ctx, tx, "23514", `
				INSERT INTO event_drafts (
					public_id, owner_user_id, episode_id, patient_id, institution_id, site_id, firm_id,
					event_type_code, intent, mode, schema_version, payload, expires_at
				) VALUES (
					'11111111-1111-4111-8111-111111111307', $1, $2, $3, $4, $5, $6,
					'core.examination', 'create', 'manual', 1,
					jsonb_build_object('note', repeat('x', 65536)), now() + interval '1 day'
				)`, fixture.ownerUserID, fixture.episodeID, fixture.patientID, fixture.institutionID, fixture.siteID, fixture.firmID)

			targetEventID := insertEvent(ctx, t, tx, fixture, "11111111-1111-4111-8111-111111111311")
			insertDraft(t, tx, fixture, "11111111-1111-4111-8111-111111111312", "update", "manual", &targetEventID)
			expectConstraintInSavepoint(t, ctx, tx, "23505", `
				INSERT INTO event_drafts (
					public_id, owner_user_id, episode_id, patient_id, institution_id, site_id, firm_id,
					event_type_code, target_event_id, intent, mode, schema_version, payload, expires_at
				) VALUES (
					'11111111-1111-4111-8111-111111111313', $1, $2, $3, $4, $5, $6,
					'core.examination', $7, 'update', 'manual', 1, '{}'::jsonb, now() + interval '1 day'
				)`, fixture.ownerUserID, fixture.episodeID, fixture.patientID, fixture.institutionID, fixture.siteID, fixture.firmID, targetEventID)

			if _, err := tx.Exec(ctx, `
				UPDATE event_drafts
				SET deleted_at = now(), deleted_by_user_id = $1, disposal_reason = 'abandoned'
				WHERE public_id = '11111111-1111-4111-8111-111111111304'`, fixture.ownerUserID); err != nil {
				t.Fatalf("dispose draft: %v", err)
			}
			expectConstraintInSavepoint(t, ctx, tx, "P0001", `
				UPDATE event_drafts SET payload = '{"changed":true}'::jsonb
				WHERE public_id = '11111111-1111-4111-8111-111111111304'`)
		})
	})

	t.Run("current event timeline indexes exist", func(t *testing.T) {
		var episodeIndex, patientIndex bool
		if err := pool.QueryRow(context.Background(), `
			SELECT
				to_regclass('public.events_episode_timeline_idx') IS NOT NULL,
				to_regclass('public.events_patient_timeline_idx') IS NOT NULL`,
		).Scan(&episodeIndex, &patientIndex); err != nil {
			t.Fatalf("query event timeline indexes: %v", err)
		}
		if !episodeIndex || !patientIndex {
			t.Fatalf("event timeline indexes present = episode:%t patient:%t, want both true", episodeIndex, patientIndex)
		}
	})
}

func expectConstraintInSavepoint(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	expectedCode string,
	query string,
	args ...any,
) {
	t.Helper()
	if _, err := tx.Exec(ctx, "SAVEPOINT expected_constraint"); err != nil {
		t.Fatalf("create constraint savepoint: %v", err)
	}
	_, err := tx.Exec(ctx, query, args...)
	expectPostgresCode(t, err, expectedCode)
	if _, err := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT expected_constraint"); err != nil {
		t.Fatalf("roll back constraint savepoint: %v", err)
	}
	if _, err := tx.Exec(ctx, "RELEASE SAVEPOINT expected_constraint"); err != nil {
		t.Fatalf("release constraint savepoint: %v", err)
	}
}

type episodeFixture struct {
	ownerUserID          int64
	patientID            int64
	institutionID        int64
	patientInstitutionID int64
	siteID               int64
	firmID               int64
	episodeID            int64
}

func insertEpisodeFixture(ctx context.Context, t *testing.T, tx pgx.Tx, key string) episodeFixture {
	t.Helper()
	fixture := episodeFixture{}
	fixture.institutionID = insertInstitution(ctx, t, tx, "Episode Hospital "+key)

	var err error
	fixture.patientID, err = insertPatient(ctx, tx, "11111111-1111-4111-8111-111111111300", "episode-"+key)
	if err != nil {
		t.Fatalf("insert episode patient: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO patient_institutions (patient_id, institution_id, association_source)
		VALUES ($1, $2, 'libreeyes')
		RETURNING id`, fixture.patientID, fixture.institutionID).Scan(&fixture.patientInstitutionID); err != nil {
		t.Fatalf("insert episode patient institution: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO sites (institution_id, name)
		VALUES ($1, $2)
		RETURNING id`, fixture.institutionID, "Episode Site "+key).Scan(&fixture.siteID); err != nil {
		t.Fatalf("insert episode site: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO firms (institution_id, name)
		VALUES ($1, $2)
		RETURNING id`, fixture.institutionID, "Episode Firm "+key).Scan(&fixture.firmID); err != nil {
		t.Fatalf("insert episode firm: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (display_name)
		VALUES ($1)
		RETURNING id`, "Episode Clinician "+key).Scan(&fixture.ownerUserID); err != nil {
		t.Fatalf("insert episode clinician: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO episodes (
			public_id, patient_id, institution_id, patient_institution_id,
			site_id, firm_id, status, started_at, created_by_user_id, updated_by_user_id
		) VALUES (
			'11111111-1111-4111-8111-111111111301', $1, $2, $3,
			$4, $5, 'open', now(), $6, $6
		)
		RETURNING id`,
		fixture.patientID,
		fixture.institutionID,
		fixture.patientInstitutionID,
		fixture.siteID,
		fixture.firmID,
		fixture.ownerUserID,
	).Scan(&fixture.episodeID); err != nil {
		t.Fatalf("insert episode: %v", err)
	}
	return fixture
}

func insertEvent(ctx context.Context, t *testing.T, tx pgx.Tx, fixture episodeFixture, publicID string) int64 {
	t.Helper()
	var eventID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO events (
			public_id, episode_id, patient_id, institution_id, patient_institution_id, site_id, firm_id,
			event_type_code, occurred_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, 'core.examination', now())
		RETURNING id`,
		publicID,
		fixture.episodeID,
		fixture.patientID,
		fixture.institutionID,
		fixture.patientInstitutionID,
		fixture.siteID,
		fixture.firmID,
	).Scan(&eventID); err != nil {
		t.Fatalf("insert event: %v", err)
	}
	return eventID
}

func insertDraft(
	t *testing.T,
	tx pgx.Tx,
	fixture episodeFixture,
	publicID string,
	intent string,
	mode string,
	targetEventID *int64,
) {
	t.Helper()
	if _, err := tx.Exec(context.Background(), `
		INSERT INTO event_drafts (
			public_id, owner_user_id, episode_id, patient_id, institution_id, site_id, firm_id,
			event_type_code, target_event_id, intent, mode, schema_version, payload, expires_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			'core.examination', $8, $9::event_draft_intent, $10::event_draft_mode, 1,
			'{}'::jsonb, now() + interval '1 day'
		)`,
		publicID,
		fixture.ownerUserID,
		fixture.episodeID,
		fixture.patientID,
		fixture.institutionID,
		fixture.siteID,
		fixture.firmID,
		targetEventID,
		intent,
		mode,
	); err != nil {
		t.Fatalf("insert draft: %v", err)
	}
}
