package main

import "testing"

// TestLoadMigrationsPairsDownFiles guards that every up migration ships a
// non-empty paired down migration, so `migrate -down` can always revert.
func TestLoadMigrationsPairsDownFiles(t *testing.T) {
	migrations, err := loadMigrations("../../migrations")
	if err != nil {
		t.Fatalf("loadMigrations: %v", err)
	}
	if len(migrations) < 17 {
		t.Fatalf("expected at least 17 migrations, got %d", len(migrations))
	}
	for _, m := range migrations {
		if m.SQL == "" {
			t.Errorf("%s: empty up SQL", m.Version)
		}
		if m.DownSQL == "" {
			t.Errorf("%s: missing paired down migration", m.Version)
		}
	}
}
