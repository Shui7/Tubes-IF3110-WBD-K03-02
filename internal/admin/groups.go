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
		SELECT g.id, g.name, COUNT(ug.user_id) AS member_count, COALESCE(g.leader_user_id, 0), COALESCE(NULLIF(u.name, ''), u.username, '')
		FROM groups g
		LEFT JOIN user_groups ug ON g.id = ug.group_id
		LEFT JOIN users u ON u.id = g.leader_user_id
		GROUP BY g.id, g.name, g.leader_user_id, u.name, u.username
		ORDER BY g.id ASC
	`)
	if err != nil {
		return nil, err
	}
	var groups []web.GroupItem
	groupIndexes := make(map[int]int)
	for rows.Next() {
		var g web.GroupItem
		if err := rows.Scan(&g.ID, &g.Name, &g.MemberCount, &g.LeaderID, &g.LeaderName); err != nil {
			rows.Close()
			return nil, err
		}
		groupIndexes[g.ID] = len(groups)
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	memberRows, err := db.Query(`
		SELECT ug.group_id, u.id, u.username, COALESCE(NULLIF(u.name, ''), u.username)
		FROM user_groups ug
		JOIN users u ON u.id = ug.user_id
		ORDER BY ug.group_id, u.id
	`)
	if err != nil {
		return nil, err
	}
	defer memberRows.Close()
	for memberRows.Next() {
		var groupID int
		var member web.GroupMemberItem
		if err := memberRows.Scan(&groupID, &member.ID, &member.Username, &member.Name); err != nil {
			return nil, err
		}
		if index, ok := groupIndexes[groupID]; ok {
			group := &groups[index]
			group.Members = append(group.Members, member)
			if member.ID == group.LeaderID {
				group.LeaderIsMember = true
			}
		}
	}
	if err := memberRows.Err(); err != nil {
		return nil, err
	}
	return groups, nil
}

func SetGroupLeader(db *sql.DB, groupID, userID int) error {
	if userID == 0 {
		_, err := db.Exec(`UPDATE groups SET leader_user_id=NULL WHERE id=$1`, groupID)
		return err
	}
	result, err := db.Exec(`
		UPDATE groups SET leader_user_id=$1
		WHERE id=$2 AND EXISTS (SELECT 1 FROM user_groups WHERE group_id=$2 AND user_id=$1)`, userID, groupID)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed == 0 {
		return sql.ErrNoRows
	}
	return nil
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

	id, err := decodeDeleteID(r)
	if err != nil {
		http.Error(w, "Invalid request: id must be a positive integer", http.StatusBadRequest)
		return
	}
	if err := DeleteGroup(db, id); err != nil {
		slog.ErrorContext(r.Context(), "admin.group_delete_failed", "error", err)
		http.Error(w, "Failed to delete group", http.StatusInternalServerError)
		return
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

func HandleSetGroupLeader(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin#groups", http.StatusSeeOther)
		return
	}
	groupID, groupErr := strconv.Atoi(r.FormValue("group_id"))
	userID, userErr := strconv.Atoi(r.FormValue("leader_user_id"))
	if groupErr == nil && userErr == nil {
		if err := SetGroupLeader(db, groupID, userID); err != nil {
			slog.ErrorContext(r.Context(), "admin.group_leader_update_failed", "group_id", groupID, "leader_user_id", userID, "error", err)
		}
	}
	http.Redirect(w, r, "/admin#groups", http.StatusSeeOther)
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

	id, err := decodeDeleteID(r)
	if err != nil {
		http.Error(w, "Invalid request: id must be a positive integer", http.StatusBadRequest)
		return
	}
	if err := DeletePermission(db, id); err != nil {
		slog.ErrorContext(r.Context(), "admin.permission_delete_failed", "error", err)
		http.Error(w, "Failed to delete permisssion", http.StatusInternalServerError)
		return
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
