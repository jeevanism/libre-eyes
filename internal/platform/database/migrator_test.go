package database

import (
	"testing"
	"testing/fstest"
)

func TestLoadMigrationsOrdersPairedFiles(t *testing.T) {
	files := fstest.MapFS{
		"000002_second.down.sql": {Data: []byte("down 2")},
		"000001_first.up.sql":    {Data: []byte("up 1")},
		"000001_first.down.sql":  {Data: []byte("down 1")},
		"000002_second.up.sql":   {Data: []byte("up 2")},
	}

	migrations, err := loadMigrations(files)
	if err != nil {
		t.Fatalf("loadMigrations() error = %v", err)
	}
	if len(migrations) != 2 || migrations[0].version != 1 || migrations[1].version != 2 {
		t.Fatalf("migration order = %#v, want versions 1 then 2", migrations)
	}
}

func TestLoadMigrationsRejectsMissingPair(t *testing.T) {
	files := fstest.MapFS{
		"000001_first.up.sql": {Data: []byte("up 1")},
	}

	if _, err := loadMigrations(files); err == nil {
		t.Fatal("loadMigrations() error = nil, want missing pair error")
	}
}
