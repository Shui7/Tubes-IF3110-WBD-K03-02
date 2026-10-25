package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/admin"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/auth"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/dbmanager"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/dbplatform"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/web"
)

var platformDB *sql.DB
var authHandler *auth.AuthHandler
var authMiddleware *auth.AuthMiddleware

func loadEnv(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 && os.Getenv(strings.TrimSpace(parts[0])) == "" {
			_ = os.Setenv(strings.TrimSpace(parts[0]), strings.Trim(strings.TrimSpace(parts[1]), `"'`))
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

	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		db, err := dbplatform.InitPlatformDB(dbURL)
		if err != nil {
			log.Printf("Database initialization failed: %v", err)
		} else {
			platformDB = db
			log.Println("Database initialized successfully")
		}
	} else {
		log.Println("DATABASE_URL is empty")
		return
	}

	sessionStore := auth.NewSessionStore(platformDB, 1*time.Hour, 24*time.Hour, 5*time.Minute)
	authHandler = auth.NewAuthHandler(platformDB, sessionStore)
	authMiddleware = auth.NewAuthMiddleware(sessionStore)

	http.HandleFunc("/", home)
	http.HandleFunc("/login", authHandler.HandleLogin)
	http.HandleFunc("/logout", logout)
	http.Handle("/admin", authMiddleware.RequireAuth(http.HandlerFunc(adminPage)))
	http.Handle("/admin/connections/add", authMiddleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin.HandleAddConnection(platformDB, w, r)
	})))
	http.Handle("/admin/connections/delete", authMiddleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin.HandleDeleteConnection(platformDB, w, r)
	})))
	http.Handle("/admin/groups/add", authMiddleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin.HandleAddGroup(platformDB, w, r)
	})))
	http.Handle("/admin/groups/delete", authMiddleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin.HandleDeleteGroup(platformDB, w, r)
	})))
	http.Handle("/admin/permissions/grant", authMiddleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin.HandleGrantPermission(platformDB, w, r)
	})))
	http.Handle("/admin/permissions/delete", authMiddleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin.HandleDeletePermission(platformDB, w, r)
	})))
	http.Handle("/admin/users/add", authMiddleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin.HandleAddUser(platformDB, w, r)
	})))
	http.Handle("/admin/users/delete", authMiddleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin.HandleDeleteUser(platformDB, w, r)
	})))
	http.Handle("/databases", authMiddleware.RequireAuth(http.HandlerFunc(databases)))
	http.Handle("/databases/", authMiddleware.RequireAuth(http.HandlerFunc(databases)))

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("Listening at http://localhost%s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

func currentUser(r *http.Request) web.BaseData {
	session, ok := r.Context().Value(auth.SessionContextKey).(*auth.Session)
	if !ok || session == nil {
		return web.BaseData{Name: "", Role: "Guest"}
	}

	return web.BaseData{
		Name:      session.Name,
		Role:      session.Role,
		CSRFToken: session.CSRFToken,
	}
}

func home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	web.RenderTemplate(w, "home.html", currentUser(r))
}

func login(w http.ResponseWriter, r *http.Request) {
	authHandler.HandleLogin(w, r)
}

func logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("session_id"); err == nil && cookie.Value != "" && platformDB != nil {
		_, _ = platformDB.Exec("UPDATE sessions SET is_revoked = TRUE WHERE id = $1", cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: "session_id", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func adminPage(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	user.Error = r.URL.Query().Get("error")
	user.Success = r.URL.Query().Get("success")
	data := web.AdminData{BaseData: user, ActiveTab: r.URL.Query().Get("tab")}
	if platformDB != nil {
		var err error
		if data.Connections, err = admin.GetConnections(platformDB); err != nil {
			log.Printf("GetConnections error: %v", err)
		}
		if data.Groups, err = admin.GetGroups(platformDB); err != nil {
			log.Printf("GetGroups error: %v", err)
		}
		if data.Users, err = admin.GetUsers(platformDB); err != nil {
			log.Printf("GetUsers error: %v", err)
		}
		if data.Permissions, err = admin.GetPermissions(platformDB); err != nil {
			log.Printf("GetPermissions error: %v", err)
		}
	}
	web.RenderTemplate(w, "admin.html", data)
}

func databases(w http.ResponseWriter, r *http.Request) {
	subPath := strings.Trim(strings.TrimPrefix(r.URL.Path, "/databases"), "/")
	if subPath == "" {
		renderDatabases(w)
		return
	}
	parts := strings.Split(subPath, "/")
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if len(parts) == 2 && parts[1] == "migrations" {
		handleMigration(id, w, r)
		return
	}
	if len(parts) != 1 || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	renderDatabaseDetail(id, w, r)
}

func renderDatabases(w http.ResponseWriter) {
	data := web.DatabasesData{BaseData: currentUser()}
	if platformDB == nil {
		data.Error = "Platform database is unavailable."
	} else if connections, err := admin.GetConnections(platformDB); err != nil {
		data.Error = fmt.Sprintf("Unable to load databases: %v", err)
	} else {
		for _, connection := range connections {
			data.Databases = append(data.Databases, web.DatabaseItem{ID: connection.ID, Name: connection.Database, AccessLevel: "edit"})
		}
	}
	web.RenderTemplate(w, "databases.html", data)
}

type managedConnection struct {
	name, host, database, username, password string
	port                                     int
}

func getManagedConnection(id int) (managedConnection, error) {
	if platformDB == nil {
		return managedConnection{}, fmt.Errorf("platform database is unavailable")
	}
	var connection managedConnection
	err := platformDB.QueryRow(`
		SELECT COALESCE(NULLIF(database_name, ''), NULLIF(name, ''), dbname, ''), host, port,
		       COALESCE(NULLIF(database_name, ''), NULLIF(name, ''), dbname, ''), username, password
		FROM database_connections WHERE id = $1`, id).Scan(
		&connection.name, &connection.host, &connection.port, &connection.database,
		&connection.username, &connection.password)
	if err == sql.ErrNoRows {
		return managedConnection{}, fmt.Errorf("database %d was not found", id)
	}
	if err != nil {
		return managedConnection{}, fmt.Errorf("load database %d: %w", id, err)
	}
	return connection, nil
}

func openManagedDatabase(ctx context.Context, id int) (*dbmanager.Manager, managedConnection, error) {
	connection, err := getManagedConnection(id)
	if err != nil {
		return nil, managedConnection{}, err
	}
	connectionString := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		quoteConnectionValue(connection.host), connection.port, quoteConnectionValue(connection.database),
		quoteConnectionValue(connection.username), quoteConnectionValue(connection.password))
	manager, err := dbmanager.Open(ctx, connectionString)
	if err != nil {
		return nil, managedConnection{}, fmt.Errorf("connect to %s: %w", connection.name, err)
	}
	return manager, connection, nil
}

func quoteConnectionValue(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "\\'") + "'"
}

func renderDatabaseDetail(id int, w http.ResponseWriter, r *http.Request) {
	manager, connection, err := openManagedDatabase(r.Context(), id)
	if err != nil {
		renderDatabaseError(w, r, id, fmt.Sprintf("Database %d", id), r.URL.Query().Get("tab"), err)
		return
	}
	defer manager.Close()

	const schema = "public"
	details, err := manager.ListTableDetails(r.Context(), schema)
	if err != nil {
		renderDatabaseError(w, r, id, connection.name, r.URL.Query().Get("tab"), err)
		return
	}
	if len(details) == 0 {
		renderDatabaseError(w, r, id, connection.name, r.URL.Query().Get("tab"), fmt.Errorf("no tables found in schema %q", schema))
		return
	}
	table := r.URL.Query().Get("table")
	if table == "" {
		table = details[0].Name
	}
	data := web.DatabaseDetailData{
		BaseData: currentUser(), Database: web.DatabaseItem{ID: id, Name: connection.name, AccessLevel: "edit"},
		SelectedTable: table, ActiveTab: r.URL.Query().Get("tab"),
	}
	var selected bool
	for _, detail := range details {
		count, countErr := manager.CountRows(r.Context(), detail.Schema, detail.Name)
		if countErr != nil {
			log.Printf("Count rows for %s.%s: %v", detail.Schema, detail.Name, countErr)
		}
		data.Tables = append(data.Tables, web.TableSummary{Name: detail.Name, RowCount: count})
		if detail.Name == table {
			selected = true
			data.Columns = convertColumns(detail.Columns)
			data.Constraints = convertConstraints(detail.Constraints)
			data.Indexes = convertIndexes(detail.Indexes)
		}
	}
	if !selected {
		renderDatabaseError(w, r, id, connection.name, r.URL.Query().Get("tab"), fmt.Errorf("table %q was not found", table))
		return
	}
	rows, err := manager.PreviewRows(r.Context(), schema, table, 100)
	if err != nil {
		renderDatabaseError(w, r, id, connection.name, r.URL.Query().Get("tab"), err)
		return
	}
	data.DataHeaders, data.DataRows = convertRows(rows, data.Columns)
	migrations, err := manager.ListMigrations(r.Context())
	if err != nil {
		renderDatabaseError(w, r, id, connection.name, r.URL.Query().Get("tab"), err)
		return
	}
	data.Migrations = convertMigrations(migrations)
	web.RenderTemplate(w, "databasedetail.html", data)
}

func renderDatabaseError(w http.ResponseWriter, r *http.Request  id int, name, tab string, err error) {
	web.RenderTemplate(w, "databasedetail.html", web.DatabaseDetailData{
		BaseData: currentUser(r), Database: web.DatabaseItem{ID: id, Name: name, AccessLevel: "view"},
		Error: err.Error(), ActiveTab: tab,
	})
}

func handleMigration(id int, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/databases/"+strconv.Itoa(id), http.StatusSeeOther)
		return
	}
	manager, _, err := openManagedDatabase(r.Context(), id)
	if err == nil {
		defer manager.Close()
		version, versionErr := manager.NextMigrationVersion(r.Context())
		err = versionErr
		if err == nil {
			err = manager.ExecuteMigration(r.Context(), version, r.FormValue("sql"), currentUser().Name)
		}
	}
	if err != nil {
		http.Redirect(w, r, "/databases/"+strconv.Itoa(id)+"?error="+url.QueryEscape(err.Error())+"#migrations", http.StatusSeeOther)
		return
	}
	message := "Migration executed successfully."
	if name := strings.TrimSpace(r.FormValue("name")); name != "" {
		message = name + " executed successfully."
	}
	http.Redirect(w, r, "/databases/"+strconv.Itoa(id)+"?success="+url.QueryEscape(message)+"#migrations", http.StatusSeeOther)
}

func convertColumns(columns []dbmanager.Column) []web.ColumnItem {
	result := make([]web.ColumnItem, 0, len(columns))
	for _, column := range columns {
		result = append(result, web.ColumnItem{Name: column.Name, Type: column.DataType, Nullable: column.IsNullable, Default: nullStringValue(column.DefaultExpr)})
	}
	return result
}

func convertConstraints(constraints []dbmanager.Constraint) []web.ConstraintItem {
	result := make([]web.ConstraintItem, 0, len(constraints))
	for _, constraint := range constraints {
		result = append(result, web.ConstraintItem{Name: constraint.Name, Type: constraint.Type, Definition: constraint.Definition})
	}
	return result
}

func convertIndexes(indexes []dbmanager.Index) []web.IndexItem {
	result := make([]web.IndexItem, 0, len(indexes))
	for _, index := range indexes {
		result = append(result, web.IndexItem{Name: index.Name, Columns: index.Definition, Type: "BTREE"})
	}
	return result
}

func convertRows(rows []dbmanager.Row, columns []web.ColumnItem) ([]string, [][]string) {
	headers := make([]string, 0, len(columns))
	for _, column := range columns {
		headers = append(headers, column.Name)
	}
	result := make([][]string, 0, len(rows))
	for _, row := range rows {
		values := make([]string, 0, len(headers))
		for _, header := range headers {
			value, ok := row[header]
			if !ok || value == nil {
				values = append(values, "")
			} else {
				values = append(values, fmt.Sprint(value))
			}
		}
		result = append(result, values)
	}
	return headers, result
}

func convertMigrations(migrations []dbmanager.Migration) []web.MigrationHistoryItem {
	result := make([]web.MigrationHistoryItem, 0, len(migrations))
	for _, migration := range migrations {
		executedAt := ""
		if migration.ExecutedAt.Valid {
			executedAt = migration.ExecutedAt.Time.Format(time.RFC3339)
		}
		version := strconv.FormatInt(migration.Version, 10)
		result = append(result, web.MigrationHistoryItem{
			Version: version, Name: "Migration " + version, ExecutedBy: migration.ExecutedBy,
			ExecutedAt: executedAt, Status: migration.Status, SQL: migration.SQL,
		})
	}
	return result
}

func nullStringValue(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}
