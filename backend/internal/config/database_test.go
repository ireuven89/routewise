package config

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestSortMigrations(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "four-digit prefixes sort after three-digit ones",
			in: []string{
				"migrations/0010_add_otp_table.sql",
				"migrations/0012_add_google_maps_integration.sql",
				"migrations/001_init_schema.sql",
				"migrations/002_schema_changes.sql",
				"migrations/009_add_company_codes.sql",
			},
			want: []string{
				"migrations/001_init_schema.sql",
				"migrations/002_schema_changes.sql",
				"migrations/009_add_company_codes.sql",
				"migrations/0010_add_otp_table.sql",
				"migrations/0012_add_google_maps_integration.sql",
			},
		},
		{
			name: "same number falls back to name order",
			in:   []string{"migrations/0014_b.sql", "migrations/0014_a.sql"},
			want: []string{"migrations/0014_a.sql", "migrations/0014_b.sql"},
		},
		{
			name: "files without a numeric prefix go last",
			in:   []string{"migrations/seed.sql", "migrations/002_x.sql", "migrations/001_y.sql"},
			want: []string{"migrations/001_y.sql", "migrations/002_x.sql", "migrations/seed.sql"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := append([]string(nil), tt.in...)
			sortMigrations(got)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSortMigrations_RepoMigrationsStartWithInitSchema(t *testing.T) {
	files, err := filepath.Glob("../../migrations/*.sql")
	if err != nil || len(files) == 0 {
		t.Fatalf("glob migrations: %v (%d files)", err, len(files))
	}
	sortMigrations(files)
	if got := filepath.Base(files[0]); got != "001_init_schema.sql" {
		t.Errorf("first migration is %s, want 001_init_schema.sql", got)
	}
}
