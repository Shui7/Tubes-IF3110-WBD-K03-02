package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/admin"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/dbplatform"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/web"
)

var platformDB *sql.DB

func loadEnv(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
			if os.Getenv(key) == "" {
				_ = os.Setenv(key, val)
			}
		}
	}
}

func main() {
	loadEnv(".env")

	staticDir := "/app/static"
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		staticDir = "internal/web/static"
	}
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		db, err := dbplatform.InitPlatformDB(dbURL)
		if err != nil {
			log.Printf("Database initialization failed: %v", err)
		} else {
			platformDB = db
			defer platformDB.Close()
			log.Println("Database initialized successfully")
		}
	}

	http.HandleFunc("/", home)
	http.HandleFunc("/login", login)
	http.HandleFunc("/logout", logout)
	http.HandleFunc("/admin", adminPage)
	http.HandleFunc("/admin/connections/add", func(w http.ResponseWriter, r *http.Request) {
		admin.HandleAddConnection(platformDB, w, r)
	})
	http.HandleFunc("/admin/connections/delete", func(w http.ResponseWriter, r *http.Request) {
		admin.HandleDeleteConnection(platformDB, w, r)
	})
	http.HandleFunc("/admin/groups/add", func(w http.ResponseWriter, r *http.Request) {
		admin.HandleAddGroup(platformDB, w, r)
	})
	http.HandleFunc("/admin/groups/delete", func(w http.ResponseWriter, r *http.Request) {
		admin.HandleDeleteGroup(platformDB, w, r)
	})
	http.HandleFunc("/admin/permissions/grant", func(w http.ResponseWriter, r *http.Request) {
		admin.HandleGrantPermission(platformDB, w, r)
	})
	http.HandleFunc("/admin/permissions/delete", func(w http.ResponseWriter, r *http.Request) {
		admin.HandleDeletePermission(platformDB, w, r)
	})
	http.HandleFunc("/admin/users/add", func(w http.ResponseWriter, r *http.Request) {
		admin.HandleAddUser(platformDB, w, r)
	})
	http.HandleFunc("/admin/users/delete", func(w http.ResponseWriter, r *http.Request) {
		admin.HandleDeleteUser(platformDB, w, r)
	})
	http.HandleFunc("/databases", databases)
	http.HandleFunc("/databases/", databases)

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("Listening at http://localhost%s\n", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func currentUser() web.BaseData {
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

func logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err == nil && cookie.Value != "" && platformDB != nil {
		_, _ = platformDB.Exec("UPDATE sessions SET is_revoked = TRUE WHERE id = $1", cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func adminPage(w http.ResponseWriter, r *http.Request) {
	tab := r.URL.Query().Get("tab")
	errQuery := r.URL.Query().Get("error")
	successQuery := r.URL.Query().Get("success")

	adminUser := currentUser()
	adminUser.Role = "Admin"
	if errQuery != "" {
		adminUser.Error = errQuery
	}
	if successQuery != "" {
		adminUser.Success = successQuery
	}

	data := web.AdminData{
		BaseData:  adminUser,
		ActiveTab: tab,
	}

	if platformDB != nil {
		if conns, err := admin.GetConnections(platformDB); err == nil {
			data.Connections = conns
		} else {
			log.Printf("GetConnections error: %v", err)
		}
		if groups, err := admin.GetGroups(platformDB); err == nil {
			data.Groups = groups
		} else {
			log.Printf("GetGroups error: %v", err)
		}
		if users, err := admin.GetUsers(platformDB); err == nil {
			data.Users = users
		} else {
			log.Printf("GetUsers error: %v", err)
		}
		if perms, err := admin.GetPermissions(platformDB); err == nil {
			data.Permissions = perms
		} else {
			log.Printf("GetPermissions error: %v", err)
		}
	}

	web.RenderTemplate(w, "admin.html", data)
}

func databases(w http.ResponseWriter, r *http.Request) {
	subPath := strings.TrimPrefix(r.URL.Path, "/databases")
	subPath = strings.Trim(subPath, "/")

	if subPath == "" {
		web.RenderTemplate(w, "databases.html", web.DatabasesData{
			BaseData: currentUser(),
			Databases: []web.DatabaseItem{
				{ID: 1, Name: "sales_db", AccessLevel: "edit"},
				{ID: 2, Name: "reports_db", AccessLevel: "view"},
			},
		})
		return
	}

	parts := strings.Split(subPath, "/")
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		http.NotFound(w, r)
		return
	}

	tab := r.URL.Query().Get("tab")
	table := r.URL.Query().Get("table")
	if table == "" {
		table = "orders"
	}

	dbName := "sales_db"
	accessLevel := "edit"
	if id == 2 {
		dbName = "reports_db"
		accessLevel = "view"
	}

	web.RenderTemplate(w, "databasedetail.html", web.DatabaseDetailData{
		BaseData: currentUser(),
		Database: web.DatabaseItem{
			ID:          id,
			Name:        dbName,
			AccessLevel: accessLevel,
		},
		Tables: []web.TableSummary{
			{Name: "orders", RowCount: 1250},
			{Name: "order_items", RowCount: 4830},
			{Name: "products", RowCount: 320},
			{Name: "customers", RowCount: 910},
		},
		SelectedTable: table,
		Columns: []web.ColumnItem{
			{Name: "id", Type: "integer", Nullable: false, Default: "nextval('orders_id_seq')", IsPK: true},
			{Name: "customer_id", Type: "integer", Nullable: false, Default: "", IsPK: false},
			{Name: "order_date", Type: "timestamp with time zone", Nullable: false, Default: "now()", IsPK: false},
			{Name: "total_amount", Type: "numeric(10,2)", Nullable: false, Default: "0.00", IsPK: false},
			{Name: "status", Type: "character varying(50)", Nullable: false, Default: "'pending'", IsPK: false},
		},
		Constraints: []web.ConstraintItem{
			{Name: "orders_pkey", Type: "PRIMARY KEY", Definition: "PRIMARY KEY (id)"},
			{Name: "fk_orders_customer", Type: "FOREIGN KEY", Definition: "FOREIGN KEY (customer_id) REFERENCES customers(id)"},
			{Name: "chk_total_amount", Type: "CHECK", Definition: "CHECK (total_amount >= 0)"},
		},
		Indexes: []web.IndexItem{
			{Name: "orders_pkey", Columns: "id", Type: "BTREE", Unique: true},
			{Name: "idx_orders_customer_id", Columns: "customer_id", Type: "BTREE", Unique: false},
			{Name: "idx_orders_date", Columns: "order_date", Type: "BTREE", Unique: false},
		},
		DataHeaders: []string{"id", "customer_id", "order_date", "total_amount", "status"},
		DataRows: [][]string{
			{"1", "101", "2026-03-01 10:14:00+00", "150.00", "completed"},
			{"2", "104", "2026-03-02 14:22:30+00", "89.50", "completed"},
			{"3", "108", "2026-03-03 09:05:12+00", "340.00", "processing"},
			{"4", "112", "2026-03-04 16:45:00+00", "25.00", "pending"},
		},
		Migrations: []web.MigrationHistoryItem{
			{Version: "001", Name: "001_initial_schema", ExecutedBy: "admin", ExecutedAt: "2026-01-10 09:00:00", Status: "success", SQL: "CREATE TABLE customers (...); CREATE TABLE orders (...);"},
			{Version: "002", Name: "002_add_order_status_index", ExecutedBy: "johndoe", ExecutedAt: "2026-02-15 11:30:00", Status: "success", SQL: "CREATE INDEX idx_orders_status ON orders(status);"},
		},
		ActiveTab: tab,
	})
}
