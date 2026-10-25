package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

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
	http.HandleFunc("/databases", databases)
	http.HandleFunc("/databases/", databases)

	log.Println("Listening at http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
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

func admin(w http.ResponseWriter, r *http.Request) {
	tab := r.URL.Query().Get("tab")

	adminUser := currentUser()
	adminUser.Role = "Admin"

	web.RenderTemplate(w, "admin.html", web.AdminData{
		BaseData:  adminUser,
		ActiveTab: tab,
		Connections: []web.DBConnectionItem{
			{ID: 1, Database: "sales_db", Host: "db", Port: 5432, User: "postgres", Status: "Connected"},
			{ID: 2, Database: "reports_db", Host: "db", Port: 5432, User: "postgres", Status: "Connected"},
			{ID: 3, Database: "archive_db", Host: "archive.internal", Port: 5432, User: "db_admin", Status: "Failed"},
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
