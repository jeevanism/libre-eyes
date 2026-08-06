package patientsummary

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestNewRepositoryRequiresDatabase(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("NewRepository(nil) error = nil")
	}
}

func TestLoadHeaderRejectsInvalidArgumentsBeforeQuery(t *testing.T) {
	repository, err := NewRepository(&unexpectedQueryDB{t: t})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name        string
		institution int64
		publicID    string
	}{
		{name: "missing institution", institution: 0, publicID: "patient"},
		{name: "missing public id", institution: 1, publicID: ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := func() error {
				_, err := repository.LoadHeader(context.Background(), test.institution, test.publicID)
				return err
			}(); !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("error = %v, want ErrInvalidRequest", err)
			}
		})
	}
}

type unexpectedQueryDB struct{ t *testing.T }

func (db *unexpectedQueryDB) QueryRow(context.Context, string, ...any) pgx.Row {
	db.t.Fatal("unexpected database query")
	return nil
}
