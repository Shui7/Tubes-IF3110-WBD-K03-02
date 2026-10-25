package dbmanager

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var identifierPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_$]*$`)

/* displaying shi */
func (m *Manager) ListSchemas(ctx context.Context) ([]Schema, error) {
	rows, err := m.db.QueryContext(ctx, `
		SELECT schema_name
		FROM information_schema.schemata
		WHERE schema_name NOT IN ('pg_catalog', 'information_schema')
		  AND schema_name NOT LIKE 'pg_toast%'
		ORDER BY schema_name`)
	if err != nil {
		return nil, fmt.Errorf("list schemas: %w", err)
	}
	defer rows.Close()

	var schemas []Schema
	for rows.Next() {
		var schema Schema
		if err := rows.Scan(&schema.Name); err != nil {
			return nil, fmt.Errorf("scan schema: %w", err)
		}
		schemas = append(schemas, schema)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schemas: %w", err)
	}
	return schemas, nil
}

func (m *Manager) ListTables(ctx context.Context, schema string) ([]Table, error) {
	if err := validateIdentifier(schema); err != nil {
		return nil, err
	}

	rows, err := m.db.QueryContext(ctx, `
		SELECT table_schema, table_name, table_type
		FROM information_schema.tables
		WHERE table_schema = $1
		ORDER BY table_name`, schema)
	if err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}
	defer rows.Close()

	var tables []Table
	for rows.Next() {
		var table Table
		if err := rows.Scan(&table.Schema, &table.Name, &table.Type); err != nil {
			return nil, fmt.Errorf("scan table: %w", err)
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tables: %w", err)
	}
	return tables, nil
}

func (m *Manager) ListColumns(ctx context.Context, schema, table string) ([]Column, error) {
	if err := validateIdentifier(schema); err != nil {
		return nil, err
	}
	if err := validateIdentifier(table); err != nil {
		return nil, err
	}

	rows, err := m.db.QueryContext(ctx, `
		SELECT table_schema, table_name, column_name, data_type,
		       is_nullable = 'YES', column_default, ordinal_position
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position`, schema, table)
	if err != nil {
		return nil, fmt.Errorf("list columns: %w", err)
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var column Column
		if err := rows.Scan(
			&column.Schema, &column.Table, &column.Name, &column.DataType,
			&column.IsNullable, &column.DefaultExpr, &column.Ordinal,
		); err != nil {
			return nil, fmt.Errorf("scan column: %w", err)
		}
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate columns: %w", err)
	}
	return columns, nil
}

func (m *Manager) ListTableDetails(ctx context.Context, schema string) ([]TableDetails, error) {
	tables, err := m.ListTables(ctx, schema)
	if err != nil {
		return nil, err
	}
	details := make([]TableDetails, 0, len(tables))
	for _, table := range tables {
		columns, err := m.ListColumns(ctx, table.Schema, table.Name)
		if err != nil {
			return nil, err
		}
		detail := TableDetails{Schema: table.Schema, Name: table.Name, Columns: columns}

		rows, err := m.db.QueryContext(ctx, `
			SELECT c.conname, c.contype, pg_get_constraintdef(c.oid)
			FROM pg_constraint c
			JOIN pg_class t ON t.oid = c.conrelid
			JOIN pg_namespace n ON n.oid = t.relnamespace
			WHERE n.nspname = $1 AND t.relname = $2
			  AND c.contype IN ('p', 'u', 'f', 'c')
			ORDER BY c.conname`, table.Schema, table.Name)
		if err != nil {
			return nil, fmt.Errorf("list table constraints: %w", err)
		}
		for rows.Next() {
			var constraint Constraint
			if err := rows.Scan(&constraint.Name, &constraint.Type, &constraint.Definition); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan table constraint: %w", err)
			}
			detail.Constraints = append(detail.Constraints, constraint)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("iterate table constraints: %w", err)
		}
		rows.Close()

		indexRows, err := m.db.QueryContext(ctx, `
			SELECT indexname, indexdef
			FROM pg_indexes
			WHERE schemaname = $1 AND tablename = $2
			ORDER BY indexname`, table.Schema, table.Name)
		if err != nil {
			return nil, fmt.Errorf("list table indexes: %w", err)
		}
		for indexRows.Next() {
			var index Index
			if err := indexRows.Scan(&index.Name, &index.Definition); err != nil {
				indexRows.Close()
				return nil, fmt.Errorf("scan table index: %w", err)
			}
			detail.Indexes = append(detail.Indexes, index)
		}
		if err := indexRows.Err(); err != nil {
			indexRows.Close()
			return nil, fmt.Errorf("iterate table indexes: %w", err)
		}
		indexRows.Close()
		details = append(details, detail)
	}
	return details, nil
}

func (m *Manager) ListRelationships(ctx context.Context, schema string) ([]Relationship, error) {
	if err := validateIdentifier(schema); err != nil {
		return nil, err
	}
	rows, err := m.db.QueryContext(ctx, `
		SELECT ns.nspname, ts.relname, asrc.attname,
		       nt.nspname, tt.relname, atgt.attname
		FROM pg_constraint c
		JOIN pg_class ts ON ts.oid = c.conrelid
		JOIN pg_namespace ns ON ns.oid = ts.relnamespace
		JOIN pg_class tt ON tt.oid = c.confrelid
		JOIN pg_namespace nt ON nt.oid = tt.relnamespace
		JOIN LATERAL unnest(c.conkey) WITH ORDINALITY src(attnum, ord) ON TRUE
		JOIN LATERAL unnest(c.confkey) WITH ORDINALITY tgt(attnum, ord) ON tgt.ord = src.ord
		JOIN pg_attribute asrc ON asrc.attrelid = ts.oid AND asrc.attnum = src.attnum
		JOIN pg_attribute atgt ON atgt.attrelid = tt.oid AND atgt.attnum = tgt.attnum
		WHERE c.contype = 'f' AND ns.nspname = $1
		ORDER BY ts.relname, c.conname`, schema)
	if err != nil {
		return nil, fmt.Errorf("list relationships: %w", err)
	}
	defer rows.Close()

	var relationships []Relationship
	for rows.Next() {
		var relationship Relationship
		if err := rows.Scan(
			&relationship.FromSchema, &relationship.FromTable, &relationship.FromColumn,
			&relationship.ToSchema, &relationship.ToTable, &relationship.ToColumn,
		); err != nil {
			return nil, fmt.Errorf("scan relationship: %w", err)
		}
		relationships = append(relationships, relationship)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate relationships: %w", err)
	}
	return relationships, nil
}

func (m *Manager) PreviewRows(ctx context.Context, schema, table string, limit int) ([]Row, error) {
	return m.PreviewRowsPage(ctx, schema, table, limit, 0)
}

func (m *Manager) PreviewRowsPage(ctx context.Context, schema, table string, limit, offset int) ([]Row, error) {
	if err := validateIdentifier(schema); err != nil {
		return nil, err
	}
	if err := validateIdentifier(table); err != nil {
		return nil, err
	}
	if limit < 1 || limit > 1000 || offset < 0 {
		return nil, errors.New("row limit must be between 1 and 1000 and offset must not be negative")
	}

	query := fmt.Sprintf("SELECT * FROM %s.%s ORDER BY ctid LIMIT %s OFFSET %s",
		quoteIdentifier(schema), quoteIdentifier(table), strconv.Itoa(limit), strconv.Itoa(offset))
	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("preview rows: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("read row columns: %w", err)
	}

	var result []Row
	for rows.Next() {
		values := make([]any, len(columns))
		destinations := make([]any, len(columns))
		for i := range values {
			destinations[i] = &values[i]
		}
		if err := rows.Scan(destinations...); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		row := make(Row, len(columns))
		for i, column := range columns {
			row[column] = displayValue(values[i])
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}
	return result, nil
}

func displayValue(value any) any {
	if bytes, ok := value.([]byte); ok {
		return string(bytes)
	}
	return value
}

func validateIdentifier(identifier string) error {
	if !identifierPattern.MatchString(identifier) {
		return fmt.Errorf("invalid PostgreSQL identifier %q", identifier)
	}
	return nil
}

func quoteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}
