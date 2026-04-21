package database

import (
	"strings"
	"testing"
)

func TestMigrationNamesContainsV1V2V3(t *testing.T) {
	names, err := MigrationNames()
	if err != nil {
		t.Fatalf("MigrationNames error = %v", err)
	}

	found := map[string]bool{"V1__baseline_schema.sql": false, "V2__search_result_release_fields.sql": false, "V3__download_task_search_result_fk.sql": false}

	for _, name := range names {
		if _, ok := found[name]; ok {
			found[name] = true
		}
	}

	for name, ok := range found {
		if !ok {
			t.Fatalf("missing migration %q", name)
		}
	}
}

func TestReadMigrationReturnsSQL(t *testing.T) {
	sql, err := ReadMigration("V1__baseline_schema.sql")
	if err != nil {
		t.Fatalf("ReadMigration error = %v", err)
	}
	if len(sql) == 0 {
		t.Fatalf("V1 migration is empty")
	}
	if !strings.Contains(sql, "CREATE TABLE IF NOT EXISTS movies") {
		t.Fatalf("V1 missing movies table")
	}
}

func TestReadMigrationV2AddsColumns(t *testing.T) {
	sql, err := ReadMigration("V2__search_result_release_fields.sql")
	if err != nil {
		t.Fatalf("ReadMigration error = %v", err)
	}
	if !strings.Contains(sql, "ADD COLUMN IF NOT EXISTS guid") {
		t.Fatalf("V2 missing guid column")
	}
}

func TestReadMigrationV3AddsFK(t *testing.T) {
	sql, err := ReadMigration("V3__download_task_search_result_fk.sql")
	if err != nil {
		t.Fatalf("ReadMigration error = %v", err)
	}
	if !strings.Contains(sql, "fk_download_tasks_search_result") {
		t.Fatalf("V3 missing FK constraint")
	}
}