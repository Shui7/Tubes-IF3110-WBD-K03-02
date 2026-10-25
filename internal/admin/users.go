package admin

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
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
		http.Redirect(w, r, "/admin#users", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/admin#users", http.StatusSeeOther)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	name := strings.TrimSpace(r.FormValue("fullname"))
	if name == "" {
		name = strings.TrimSpace(r.FormValue("name"))
	}
	if name == "" {
		name = username
	}
	password := r.FormValue("password")
	role := strings.TrimSpace(r.FormValue("role"))
	groups := r.Form["groups"]

	if username != "" && password != "" {
		if err := AddUser(db, username, name, password, role, groups); err != nil {
			log.Printf("Error adding user: %v", err)
		}
	}

	http.Redirect(w, r, "/admin#users", http.StatusSeeOther)
}

func HandleDeleteUser(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin#users", http.StatusSeeOther)
		return
	}

	idStr := r.FormValue("id")
	id, err := strconv.Atoi(idStr)
	if err == nil {
		if err := DeleteUser(db, id); err != nil {
			log.Printf("Error deleting user: %v", err)
		}
	}

	http.Redirect(w, r, "/admin#users", http.StatusSeeOther)
}
