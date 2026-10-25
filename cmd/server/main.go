package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/web"
)

func main() {
	staticDir := "/app/static"
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		staticDir = "internal/web/static"
	}
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	http.HandleFunc("/", home)
	http.HandleFunc("/login", login)
	http.HandleFunc("/admin", admin)
	http.HandleFunc("/databases/", databases)

	log.Println("Listening at http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func currentUser() web.BaseData {
	// placeholder
	return web.BaseData{
		Name: "John User",
		Role: "Admin",
	}
}

func home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	web.RenderTemplate(w, "home.html", currentUser())
}

func login(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// Placeholder for login authentication logic
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	web.RenderTemplate(w, "login.html", web.LoginData{
		BaseData: web.BaseData{
			Name: "",
			Role: "Guest",
		},
	})
}

func admin(w http.ResponseWriter, r *http.Request) {
	tab := r.URL.Query().Get("tab")
	if tab == "" && r.URL.Path == "/admin/connections" {
		tab = "connections"
	}

	adminUser := currentUser()
	adminUser.Role = "Admin"

	web.RenderTemplate(w, "admin.html", web.AdminData{
		BaseData:  adminUser,
		ActiveTab: tab,
		Connections: []web.DBConnectionItem{
			{ID: 1, Host: "db", Port: 5432, Database: "sales_db", User: "postgres", Status: "Connected"},
			{ID: 2, Host: "db", Port: 5432, Database: "reports_db", User: "postgres", Status: "Connected"},
			{ID: 3, Host: "archive.internal", Port: 5432, Database: "archive_db", User: "db_admin", Status: "Failed"},
		},
		Groups: []web.GroupItem{
			{ID: 1, Name: "engineering", MemberCount: 3},
			{ID: 2, Name: "data-analysts", MemberCount: 5},
			{ID: 3, Name: "auditors", MemberCount: 2},
		},
		Users: []web.UserItem{
			{ID: 1, Username: "admin", Name: "System Administrator", Role: "Admin", Groups: []string{"engineering"}},
			{ID: 2, Username: "johndoe", Name: "John Doe", Role: "User", Groups: []string{"engineering", "data-analysts"}},
			{ID: 3, Username: "janesmith", Name: "Jane Smith", Role: "User", Groups: []string{"data-analysts"}},
			{ID: 4, Username: "alicedev", Name: "Alice Bob", Role: "User", Groups: []string{"auditors"}},
		},
		Permissions: []web.GroupPermissionItem{
			{ID: 1, GroupID: 1, GroupName: "engineering", DatabaseID: 1, DatabaseName: "sales_db", AccessLevel: "edit"},
			{ID: 2, GroupID: 1, GroupName: "engineering", DatabaseID: 2, DatabaseName: "reports_db", AccessLevel: "edit"},
			{ID: 3, GroupID: 2, GroupName: "data-analysts", DatabaseID: 1, DatabaseName: "sales_db", AccessLevel: "view"},
			{ID: 4, GroupID: 2, GroupName: "data-analysts", DatabaseID: 2, DatabaseName: "reports_db", AccessLevel: "view"},
			{ID: 5, GroupID: 3, GroupName: "auditors", DatabaseID: 1, DatabaseName: "sales_db", AccessLevel: "view"},
		},
	})
}

func databases(w http.ResponseWriter, r *http.Request) {
	// placeholder
	web.RenderTemplate(w, "databases.html", web.DatabasesData{
		BaseData: currentUser(),
		Databases: []web.DatabaseItem{
			{ID: 1, Name: "sales_db", AccessLevel: "edit"},
			{ID: 2, Name: "reports_db", AccessLevel: "view"},
		},
	})
}
