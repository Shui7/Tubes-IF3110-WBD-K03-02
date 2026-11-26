package jobs

import (
	"strings"
	"testing"
)

func TestValidateBatchRequest(t *testing.T) {
	const cursorSQL = `UPDATE t SET x = 1 WHERE id > COALESCE($1::bigint, 0) AND id <= COALESCE($1::bigint, 0) + $2::int`
	const attributeSQL = `UPDATE big_items SET price = price * 1.1 WHERE {{batch_filter}}`

	tests := []struct {
		name    string
		mode    string
		size    int
		sql     string
		table   string
		column  string
		wantErr string // substring, "" means no error
	}{
		{"plain ignores batch settings", ModePlain, 0, "SELECT 1", "", "", ""},

		{"custom cursor ok", ModeBatch, 1000, cursorSQL, "", "", ""},
		{"custom cursor size one", ModeBatch, 1, cursorSQL, "", "", ""},
		{"custom cursor max size", ModeBatch, MaxBatchSize, cursorSQL, "", "", ""},
		{"batch without size", ModeBatch, 0, cursorSQL, "", "", "batch size"},
		{"batch negative size", ModeBatch, -5, cursorSQL, "", "", "batch size"},
		{"batch size too large", ModeBatch, MaxBatchSize + 1, cursorSQL, "", "", "batch size"},
		{"custom cursor without $1", ModeBatch, 10, "UPDATE t SET x = 1 LIMIT $2", "", "", "$1"},
		{"custom cursor without $2", ModeBatch, 10, "UPDATE t SET x = 1 WHERE id > $1", "", "", "$2"},
		{"unknown mode", "turbo", 10, cursorSQL, "", "", "unknown migration mode"},

		{"attribute ok", ModeBatch, 10000, attributeSQL, "big_items", "id", ""},
		{"attribute with schema", ModeBatch, 10000, attributeSQL, "public.big_items", "id", ""},
		{"attribute needs placeholder", ModeBatch, 10000, "UPDATE big_items SET price = 0", "big_items", "id", "{{batch_filter}}"},
		{"attribute column without table", ModeBatch, 10000, attributeSQL, "", "id", "both a table and a column"},
		{"attribute table without column", ModeBatch, 10000, attributeSQL, "big_items", "", "both a table and a column"},
		{"attribute column with dot", ModeBatch, 10000, attributeSQL, "big_items", "t.id", "plain column name"},
		{"attribute table with too many parts", ModeBatch, 10000, attributeSQL, "a.b.c", "id", "schema.table"},
		{"attribute still needs a valid size", ModeBatch, 0, attributeSQL, "big_items", "id", "batch size"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBatchRequest(tt.mode, tt.size, tt.sql, tt.table, tt.column)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestJobBatchByAttribute(t *testing.T) {
	tests := []struct {
		table, column string
		want          bool
	}{
		{"big_items", "id", true},
		{"", "", false},
		{"big_items", "", false},
		{"", "id", false},
	}
	for _, tt := range tests {
		if got := (Job{BatchTable: tt.table, BatchColumn: tt.column}).BatchByAttribute(); got != tt.want {
			t.Errorf("BatchByAttribute(%q, %q) = %v, want %v", tt.table, tt.column, got, tt.want)
		}
	}
}
