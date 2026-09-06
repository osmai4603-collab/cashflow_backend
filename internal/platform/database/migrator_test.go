package database_test

import (
	"testing"

	"cashflow_backend/internal/platform/database"
)

func TestParseMigrationFilename(t *testing.T) {
	tests := []struct {
		filename      string
		wantVersion   int64
		wantName      string
		wantDirection string
		wantErr       bool
	}{
		{
			filename:      "000001_init_schema.up.sql",
			wantVersion:   1,
			wantName:      "init_schema",
			wantDirection: "up",
			wantErr:       false,
		},
		{
			filename:      "000002_create_partners.down.sql",
			wantVersion:   2,
			wantName:      "create_partners",
			wantDirection: "down",
			wantErr:       false,
		},
		{
			filename: "invalid_name.sql",
			wantErr:  true,
		},
		{
			filename: "abc_not_a_number.up.sql",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			mf, err := database.ParseMigrationFilename(tt.filename)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %s", tt.filename)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if mf.Version != tt.wantVersion {
				t.Errorf("got Version %d, want %d", mf.Version, tt.wantVersion)
			}
			if mf.Name != tt.wantName {
				t.Errorf("got Name %q, want %q", mf.Name, tt.wantName)
			}
			if mf.Direction != tt.wantDirection {
				t.Errorf("got Direction %q, want %q", mf.Direction, tt.wantDirection)
			}
		})
	}
}
