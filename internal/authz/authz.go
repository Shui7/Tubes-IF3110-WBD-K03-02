// Package authz implements authorization logic for the database management platform.
// Per spec: access control rules are intentionally separated from authentication
// so that permission checks, access management, and auditing can be done in a structured way.
package authz

import (
	"database/sql"
)

type AccessLevel string

const (
	AccessNone AccessLevel = "none"
	AccessView AccessLevel = "view"
	AccessEdit AccessLevel = "edit"
)

func IsAdmin(role string) bool {
	return role == "Admin"
}

type DatabaseAccess struct {
	ConnectionID int
	Name         string
	AccessLevel  AccessLevel
}

func GetUserDatabaseAccess(db *sql.DB, userID, databaseID int) (AccessLevel, error) {
	var isAdmin bool
	err := db.QueryRow(`SELECT is_admin OR (role = 'Admin') FROM users WHERE id = $1`, userID).Scan(&isAdmin)
	if err != nil {
		return AccessNone, err
	}
	if isAdmin {
		return AccessEdit, nil
	}

	var level string
	err = db.QueryRow(`
		SELECT CASE
			WHEN bool_or(gda.access_level = 'edit') THEN 'edit'
			WHEN bool_or(gda.access_level = 'view') THEN 'view'
			ELSE 'none'
		END
		FROM group_database_access gda
		JOIN user_groups ug ON ug.group_id = gda.group_id
		WHERE ug.user_id = $1
		  AND COALESCE(gda.database_id, gda.db_id) = $2
	`, userID, databaseID).Scan(&level)

	if err == sql.ErrNoRows || level == "" || level == "none" {
		return AccessNone, nil
	}
	if err != nil {
		return AccessNone, err
	}
	return AccessLevel(level), nil
}

func GetUserDatabaseList(db *sql.DB, userID int) ([]DatabaseAccess, error) {
	var isAdmin bool
	err := db.QueryRow(`SELECT is_admin OR (role = 'Admin') FROM users WHERE id = $1`, userID).Scan(&isAdmin)
	if err != nil {
		return nil, err
	}

	if isAdmin {
		rows, err := db.Query(`
			SELECT id, COALESCE(NULLIF(database_name, ''), NULLIF(name, ''), dbname, '')
			FROM database_connections ORDER BY id ASC
		`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var result []DatabaseAccess
		for rows.Next() {
			var d DatabaseAccess
			if err := rows.Scan(&d.ConnectionID, &d.Name); err != nil {
				return nil, err
			}
			d.AccessLevel = AccessEdit
			result = append(result, d)
		}
		return result, rows.Err()
	}

	rows, err := db.Query(`
		SELECT
			COALESCE(gda.database_id, gda.db_id) AS conn_id,
			COALESCE(NULLIF(dc.database_name, ''), NULLIF(dc.name, ''), dc.dbname, '') AS db_name,
			CASE
				WHEN bool_or(gda.access_level = 'edit') THEN 'edit'
				ELSE 'view'
			END AS effective_access
		FROM group_database_access gda
		JOIN user_groups ug ON ug.group_id = gda.group_id
		JOIN database_connections dc ON dc.id = COALESCE(gda.database_id, gda.db_id)
		WHERE ug.user_id = $1
		  AND COALESCE(gda.database_id, gda.db_id) IS NOT NULL
		GROUP BY COALESCE(gda.database_id, gda.db_id), dc.database_name, dc.name, dc.dbname
		ORDER BY COALESCE(gda.database_id, gda.db_id) ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []DatabaseAccess
	for rows.Next() {
		var d DatabaseAccess
		var level string
		if err := rows.Scan(&d.ConnectionID, &d.Name, &level); err != nil {
			return nil, err
		}
		d.AccessLevel = AccessLevel(level)
		result = append(result, d)
	}
	return result, rows.Err()
}
