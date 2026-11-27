package admin

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/web"
	"golang.org/x/crypto/bcrypt"
)

func GetUsers(db *sql.DB) ([]web.UserItem, error) {
	rows, err := db.Query(`
		SELECT u.id, u.username, 
		       COALESCE(NULLIF(u.name, ''), u.username) AS name, 
		       COALESCE(NULLIF(u.role, ''), CASE WHEN u.is_admin THEN 'Admin' ELSE 'User' END) AS role, 
		       COALESCE(g.name, '')
		FROM users u
		LEFT JOIN user_groups ug ON u.id = ug.user_id
		LEFT JOIN groups g ON ug.group_id = g.id
		ORDER BY u.id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	userMap := make(map[int]*web.UserItem)
	var userOrder []int

	for rows.Next() {
		var (
			id        int
			username  string
			name      string
			role      string
			groupName string
		)
		if err := rows.Scan(&id, &username, &name, &role, &groupName); err != nil {
			return nil, err
		}

		if _, exists := userMap[id]; !exists {
			userMap[id] = &web.UserItem{
				ID:       id,
				Username: username,
				Name:     name,
				Role:     role,
				Groups:   []string{},
			}
			userOrder = append(userOrder, id)
		}

		if groupName != "" {
			userMap[id].Groups = append(userMap[id].Groups, groupName)
		}
	}

	var users []web.UserItem
	for _, id := range userOrder {
		users = append(users, *userMap[id])
	}
	return users, rows.Err()
}

func AddUser(db *sql.DB, username, name, password, role string, groupNames []string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	isAdmin := (role == "Admin")
	if !isAdmin {
		role = "User"
	}
	if name == "" {
		name = username
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var userID int
	err = tx.QueryRow(`
		INSERT INTO users (username, password_hash, password, name, role, is_admin) 
		VALUES ($1, $2, $2, $3, $4, $5) 
		RETURNING id
	`, username, string(hashedPassword), name, role, isAdmin).Scan(&userID)
	if err != nil {
		return err
	}

	for _, groupName := range groupNames {
		var groupID int
		err := tx.QueryRow("SELECT id FROM groups WHERE name = $1", groupName).Scan(&groupID)
		if err == nil {
			_, _ = tx.Exec("INSERT INTO user_groups (user_id, group_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", userID, groupID)
		}
	}

	return tx.Commit()
}

func DeleteUser(db *sql.DB, id int) error {
	_, err := db.Exec("DELETE FROM users WHERE id = $1", id)
	return err
}

func HandleAddUser(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type addUserRequest struct {
		Username string   `json:"username"`
		Fullname string   `json:"fullname"`
		Password string   `json:"password"`
		Role     string   `json:"role"`
		Groups   []string `json:"groups"`
	}
	var req addUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	username := strings.TrimSpace(req.Username)
	name := strings.TrimSpace(req.Fullname)
	if name == "" {
		name = username
	}
	password := req.Password
	role := strings.TrimSpace(req.Role)
	groups := req.Groups

	if username != "" && password != "" {
		if err := AddUser(db, username, name, password, role, groups); err != nil {
			slog.ErrorContext(r.Context(), "admin.user_add_failed", "error", err)
			http.Error(w, "Failed to add user", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func HandleDeleteUser(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := decodeDeleteID(r)
	if err != nil {
		http.Error(w, "Invalid request: id must be a positive integer", http.StatusBadRequest)
		return
	}
	if err := DeleteUser(db, id); err != nil {
		slog.ErrorContext(r.Context(), "admin.user_delete_failed", "error", err)
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func HandleGetUsers(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	users, err := GetUsers(db)
	if err != nil {
		slog.ErrorContext(r.Context(), "admin.user_fetch_failed", "error", err)
		http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}
