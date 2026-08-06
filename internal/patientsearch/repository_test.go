package patientsearch

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestNewRepositoryRequiresDatabase(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("NewRepository(nil) error = nil")
	}
}

func TestRepositoryRejectsInvalidArgumentsBeforeQuery(t *testing.T) {
	database := &unexpectedQueryDB{t: t}
	repository, err := NewRepository(database)
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	validDate := time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
	validCriteria := DemographicCriteria{
		FamilyNameNormalized: "patient",
		DateOfBirth:          validDate,
		Gender:               GenderUnknown,
	}

	tests := []struct {
		name string
		run  func() error
	}{
		{name: "identifier institution", run: func() error {
			_, err := repository.SearchByIdentifier(context.Background(), 0, 1, 1, "A1", 25, nil)
			return err
		}},
		{name: "identifier empty canonical", run: func() error {
			_, err := repository.SearchByIdentifier(context.Background(), 1, 1, 1, "", 25, nil)
			return err
		}},
		{name: "zero limit", run: func() error {
			_, err := repository.SearchByDemographics(context.Background(), 1, 1, validCriteria, 0, nil)
			return err
		}},
		{name: "excessive limit", run: func() error {
			_, err := repository.SearchByDemographics(context.Background(), 1, 1, validCriteria, 101, nil)
			return err
		}},
		{name: "invalid gender", run: func() error {
			criteria := validCriteria
			criteria.Gender = "invalid"
			_, err := repository.SearchByDemographics(context.Background(), 1, 1, criteria, 25, nil)
			return err
		}},
		{name: "incomplete boundary", run: func() error {
			_, err := repository.SearchByDemographics(context.Background(), 1, 1, validCriteria, 25, &PageBoundary{
				DateOfBirth: time.Time{}, PublicID: "not-empty",
			})
			return err
		}},
		{name: "duplicate missing name", run: func() error {
			_, err := repository.FindDemographicDuplicates(context.Background(), 1, 1, DuplicateDemographicCriteria{
				GivenNameNormalized: "given", DateOfBirth: validDate,
			})
			return err
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.run(); !errors.Is(err, ErrInvalidRepositoryRequest) {
				t.Fatalf("repository error = %v, want invalid request", err)
			}
		})
	}
}

type unexpectedQueryDB struct {
	t *testing.T
}

func (db *unexpectedQueryDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	db.t.Fatal("unexpected database query")
	return nil, errors.New("unreachable")
}

func (db *unexpectedQueryDB) QueryRow(context.Context, string, ...any) pgx.Row {
	db.t.Fatal("unexpected database query row")
	return nil
}
