package admin

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/web"
)

func GetGroups(db *sql.DB) ([]web.GroupItem, error) {
	rows, err := db.Query(`
		SELECT g.id, g.name, COUNT(ug.user_id) AS member_count
		FROM groups g
		LEFT JOIN user_groups ug ON g.id = ug.group_id
		GROUP BY g.id, g.name
		ORDER BY g.id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []web.GroupItem
	for rows.Next() {
		var g web.GroupItem
		if err := rows.Scan(&g.ID, &g.Name, &g.MemberCount); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

func AddGroup(db *sql.DB, name string) error {
	_, err := db.Exec("INSERT INTO groups (name) VALUES ($1)", name)
	return err
}

func DeleteGroup(db *sql.DB, id int) error {
	_, err := db.Exec("DELETE FROM groups WHERE id = $1", id)
	return err
}

func GetPermissions(db *sql.DB) ([]web.GroupPermissionItem, error) {
	rows, err := db.Query(`
		SELECT gda.id, gda.group_id, g.name, 
		       COALESCE(gda.database_id, gda.db_id, dc.id), 
		       COALESCE(NULLIF(dc.database_name, ''), NULLIF(dc.name, ''), dc.dbname, ''), 
		       COALESCE(NULLIF(gda.access_level, ''), 'view')
		FROM group_database_access gda
		JOIN groups g ON gda.group_id = g.id
		JOIN database_connections dc ON (gda.database_id = dc.id OR gda.db_id = dc.id)
		ORDER BY gda.id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []web.GroupPermissionItem
	for rows.Next() {
		var p web.GroupPermissionItem
		if err := rows.Scan(&p.ID, &p.GroupID, &p.GroupName, &p.DatabaseID, &p.DatabaseName, &p.AccessLevel); err != nil {
			return nil, err
		}
		permissions = append(permissions, p)
	}
	return permissions, rows.Err()
}

func GrantPermission(db *sql.DB, groupID, databaseID int, accessLevel string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, _ = tx.Exec("DELETE FROM group_database_access WHERE group_id = $1 AND (database_id = $2 OR db_id = $2)", groupID, databaseID)
	_, err = tx.Exec(`
		INSERT INTO group_database_access (group_id, database_id, db_id, access_level)
		VALUES ($1, $2, $2, $3)
	`, groupID, databaseID, accessLevel)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func DeletePermission(db *sql.DB, id int) error {
	_, err := db.Exec("DELETE FROM group_database_access WHERE id = $1", id)
	return err
}

func HandleAddGroup(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type addGroupRequest struct {
		Name string `json:"name"`
	}
	var req addGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(req.Name)
	if name != "" {
		if err := AddGroup(db, name); err != nil {
			slog.ErrorContext(r.Context(), "admin.group_add_failed", "error", err)
			http.Error(w, "Failed to create group", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func HandleDeleteGroup(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type deleteGroupRequest struct {
		ID string `json:"id"`
	}
	var req deleteGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	idStr := req.ID
	id, err := strconv.Atoi(idStr)
	if err == nil {
		if err := DeleteGroup(db, id); err != nil {
			slog.ErrorContext(r.Context(), "admin.group_delete_failed", "error", err)
			http.Error(w, "Failed to delete group", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func HandleGetGroups(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	groups, err := GetGroups(db)
	if err != nil {
		slog.ErrorContext(r.Context(), "admin.group_fetch_failed", "error", err)
		http.Error(w, "Failed to fetch groups", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(groups)
}

func HandleGrantPermission(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type grantPermissionRequest struct {
		GroupID     string `json:"group_id"`
		DatabaseID  string `json:"database_id"`
		AccessLevel string `json:"access_level"`
	}
	var req grantPermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	groupIDStr := req.GroupID
	databaseIDStr := req.DatabaseID
	accessLevel := strings.TrimSpace(req.AccessLevel)

	groupID, err1 := strconv.Atoi(groupIDStr)
	databaseID, err2 := strconv.Atoi(databaseIDStr)

	if err1 == nil && err2 == nil && (accessLevel == "view" || accessLevel == "edit") {
		if err := GrantPermission(db, groupID, databaseID, accessLevel); err != nil {
			slog.ErrorContext(r.Context(), "admin.permission_grant_failed", "error", err)
			http.Error(w, "Failed to grant permisssion", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func HandleDeletePermission(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type deletePermissionRequest struct {
		ID string `json:"id"`
	}
	var req deletePermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	idStr := req.ID
	id, err := strconv.Atoi(idStr)
	if err == nil {
		if err := DeletePermission(db, id); err != nil {
			slog.ErrorContext(r.Context(), "admin.permission_delete_failed", "error", err)
			http.Error(w, "Failed to delete permisssion", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func HandleGetPermissions(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	permissions, err := GetPermissions(db)
	if err != nil {
		slog.ErrorContext(r.Context(), "admin.permission_fetch_failed", "error", err)
		http.Error(w, "Failed to fetch permissions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(permissions)
}
