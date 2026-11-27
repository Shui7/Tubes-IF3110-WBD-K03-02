package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/admin"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/auth"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/authz"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/dbmanager"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/dbplatform"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/jobs"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/observability"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/web"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/workflow"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var platformDB *sql.DB
var sessionStore *auth.SessionStore
var authHandlerAPI *auth.AuthHandlerAPI
var jobStore *jobs.Store
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

	ctx := context.Background()
	logger := observability.SetupLogger(observability.LoggerConfig{Service: "app"})
	shutdownTracing, err := observability.SetupTracing(ctx, observability.TracingConfig{
		ServiceName: "app",
		Insecure:    true, // only used when OTEL_EXPORTER=otlp
	})
	if err != nil {
		logger.ErrorContext(ctx, "tracing.setup_failed", "error", err)
		os.Exit(1)
	}
	defer func() { _ = shutdownTracing(context.Background()) }()

	staticDir := "/app/static"
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		staticDir = "internal/web/static"
	}
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		db, err := dbplatform.InitPlatformDB(dbURL)
		if err != nil {
			slog.Error("platformdb.init_failed", "error", err)
		} else {
			platformDB = db
			slog.Info("platformdb.initialized")
			if err := jobs.EnsureSchema(ctx, db); err != nil {
				slog.ErrorContext(ctx, "jobs.schema_failed", "error", err)
				return
			}
			jobStore = jobs.NewStore(db).WithMaxPending(envPositiveInt("MAX_PENDING_JOBS_PER_DB", jobs.DefaultMaxPendingPerDatabase))
		}
	} else {
		slog.Error("config.missing_database_url")
		return
	}

	sessionStore = auth.NewSessionStore(platformDB, 1*time.Hour, 24*time.Hour, 5*time.Minute)
	authHandler = auth.NewAuthHandler(platformDB, sessionStore)
	authHandlerAPI = auth.NewAuthHandlerAPI(platformDB, sessionStore)
	authMiddleware = auth.NewAuthMiddleware(sessionStore)
	adminOnly := func(next http.Handler) http.Handler {
		return authMiddleware.RequireAuth(authMiddleware.RequireRole("Admin", next))
	}

	http.HandleFunc("/", index)
	http.HandleFunc("/health", health)
	http.HandleFunc("/api/auth/login", authHandlerAPI.HandleLogin)
	http.HandleFunc("/api/auth/logout", authHandlerAPI.HandleLogout)
	http.HandleFunc("/api/auth/me", authHandlerAPI.GetSessionInfo)
	http.HandleFunc("/api/auth/refresh-login", authHandlerAPI.RefreshLoginCsrfToken)
	http.Handle("/api/admin/connections", adminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin.HandleGetConnections(platformDB, w, r)
	})))
	http.Handle("/api/admin/connections/add", adminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin.HandleAddConnection(platformDB, w, r)
	})))
	http.Handle("/api/admin/connections/delete", adminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin.HandleDeleteConnection(platformDB, w, r)
	})))
	http.Handle("/api/admin/groups", adminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin.HandleGetGroups(platformDB, w, r)
	})))
	http.Handle("/api/admin/groups/add", adminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin.HandleAddGroup(platformDB, w, r)
	})))
	http.Handle("/api/admin/groups/delete", adminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin.HandleDeleteGroup(platformDB, w, r)
	})))
	http.Handle("/api/admin/permissions", adminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin.HandleGetPermissions(platformDB, w, r)
	})))
	http.Handle("/admin/groups/leader", adminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin.HandleSetGroupLeader(platformDB, w, r)
	})))
	http.Handle("/api/admin/permissions/grant", adminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin.HandleGrantPermission(platformDB, w, r)
	})))
	http.Handle("/api/admin/permissions/delete", adminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin.HandleDeletePermission(platformDB, w, r)
	})))
	http.Handle("/api/admin/users", adminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin.HandleGetUsers(platformDB, w, r)
	})))
	http.Handle("/api/admin/users/add", adminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin.HandleAddUser(platformDB, w, r)
	})))
	http.Handle("/api/admin/users/delete", adminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin.HandleDeleteUser(platformDB, w, r)
	})))
	http.Handle("GET /api/jobs/{id}", authMiddleware.RequireAuth(jobs.StatusHandler(jobStore, canViewDatabase)))
	http.Handle("POST /api/jobs/{id}/resume", authMiddleware.RequireAuth(jobs.ResumeHandler(jobStore, platformDB)))
	http.Handle("GET /api/databases/{id}/migrations/visibility", authMiddleware.RequireAuth(http.HandlerFunc(migrationVisibility)))
	http.Handle("GET /api/databases", authMiddleware.RequireAuth(http.HandlerFunc(apiDatabases)))
	http.Handle("GET /api/databases/{id}", authMiddleware.RequireAuth(http.HandlerFunc(apiDatabaseDetail)))
	http.HandleFunc("GET /databases", index)
	http.HandleFunc("GET /databases/", index)
	http.Handle("POST /databases/", authMiddleware.RequireAuth(http.HandlerFunc(databases)))

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	srv := &http.Server{
		Addr:    addr,
		Handler: otelhttp.NewHandler(observability.RequestIDMiddleware(http.DefaultServeMux), "http.server"),
	}
	slog.InfoContext(ctx, "server.listening", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.ErrorContext(ctx, "server.failed", "error", err)
		_ = shutdownTracing(ctx) // os.Exit skips defers, so flush spans explicitly
		os.Exit(1)
	}
}

func index(w http.ResponseWriter, r *http.Request) {
	csrfToken := ""
	cookie, err := r.Cookie("session_id")
	if err == nil && cookie != nil {
		session, err := sessionStore.GetSession(cookie.Value)
		if err == nil && session != nil {
			_ = sessionStore.TouchSession(session.ID)
			csrfToken = session.CSRFToken
		}
	}
	web.RenderTemplate(w, "index.html", web.BaseData{
		Name:      "Guest",
		Role:      "Guest",
		CSRFToken: csrfToken,
	})

}

func unauthorizedPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	web.RenderTemplate(w, "unauthorized.html", web.BaseData{
		Name: "Guest",
		Role: "Guest",
	})
}

func currentUser(r *http.Request) web.BaseData {
	session, ok := r.Context().Value(auth.SessionContextKey).(*auth.Session)
	if !ok || session == nil {
		return web.BaseData{Name: "Guest", Role: "Guest"}
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
	if authHandler == nil {
		http.Error(w, "authentication unavailable", http.StatusInternalServerError)
		return
	}
	authHandler.HandleLogout(w, r)
}

func adminPage(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	user.Error = r.URL.Query().Get("error")
	user.Success = r.URL.Query().Get("success")
	data := web.AdminData{BaseData: user, ActiveTab: r.URL.Query().Get("tab")}
	if platformDB != nil {
		var err error
		if data.Connections, err = admin.GetConnections(platformDB); err != nil {
			slog.ErrorContext(r.Context(), "admin.load_failed", "resource", "connections", "error", err)
		}
		if data.Groups, err = admin.GetGroups(platformDB); err != nil {
			slog.ErrorContext(r.Context(), "admin.load_failed", "resource", "groups", "error", err)
		}
		if data.Users, err = admin.GetUsers(platformDB); err != nil {
			slog.ErrorContext(r.Context(), "admin.load_failed", "resource", "users", "error", err)
		}
		if data.Permissions, err = admin.GetPermissions(platformDB); err != nil {
			slog.ErrorContext(r.Context(), "admin.load_failed", "resource", "permissions", "error", err)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func databases(w http.ResponseWriter, r *http.Request) {
	subPath := strings.Trim(strings.TrimPrefix(r.URL.Path, "/databases"), "/")
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	parts := strings.Split(subPath, "/")
	if len(parts) != 2 || parts[1] != "migrations" {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		http.NotFound(w, r)
		return
	}

	sess, _ := r.Context().Value(auth.SessionContextKey).(*auth.Session)
	level, err := authz.GetUserDatabaseAccess(r.Context(), platformDB, sess.UserID, id)
	if err != nil || level == authz.AccessNone {
		http.Error(w, "Forbidden: you do not have access to this database", http.StatusForbidden)
		return
	}
	if level != authz.AccessEdit {
		http.Error(w, "Forbidden: edit access required to execute migrations", http.StatusForbidden)
		return
	}

	if len(parts) >= 2 && parts[1] == "migrations" {
		// Only View+Edit users may execute migrations (POST). View-only users can still
		// reach this route via GET (which just redirects back to the detail page).
		isApprovalDecision := len(parts) == 5 && parts[2] == "requests" && parts[4] == "decision"
		if r.Method == http.MethodPost && level != authz.AccessEdit && !isApprovalDecision {
			http.Error(w, "Forbidden: edit access required to execute migrations", http.StatusForbidden)
			return
		}
		if isApprovalDecision {
			requestID, parseErr := strconv.ParseInt(parts[3], 10, 64)
			if parseErr != nil {
				http.NotFound(w, r)
				return
			}
			handleMigrationDecision(id, requestID, w, r)
			return
		}
		if len(parts) > 3 || (len(parts) == 3 && parts[2] != "rollback" && parts[2] != "forward") {
			http.NotFound(w, r)
			return
		}
		operation := ""
		if len(parts) == 3 {
			operation = parts[2]
		}
		handleMigration(id, operation, w, r)
		return
	}
	handleMigration(id, "", w, r)
}

func apiDatabases(w http.ResponseWriter, r *http.Request) {
	data := web.DatabasesData{BaseData: currentUser(r)}
	if platformDB == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Platform database is unavailable."})
		return
	}
	sess, _ := r.Context().Value(auth.SessionContextKey).(*auth.Session)
	dbList, err := authz.GetUserDatabaseList(platformDB, sess.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Unable to load databases: %v", err)})
		return
	}
	for _, d := range dbList {
		data.Databases = append(data.Databases, web.DatabaseItem{
			ID: d.ConnectionID, Name: d.Name, AccessLevel: string(d.AccessLevel),
		})
	}
	writeJSON(w, http.StatusOK, data)
}

func apiDatabaseDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid database ID"})
		return
	}
	if platformDB == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Platform database is unavailable."})
		return
	}
	sess, _ := r.Context().Value(auth.SessionContextKey).(*auth.Session)
	level, err := authz.GetUserDatabaseAccess(r.Context(), platformDB, sess.UserID, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Unable to check database access"})
		return
	}
	if level == authz.AccessNone {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "You do not have access to this database"})
		return
	}
	data, err := loadDatabaseDetail(id, string(level), r)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("api.response_encode_failed", "error", err)
	}
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

func loadDatabaseDetail(id int, accessLevel string, r *http.Request) (web.DatabaseDetailData, error) {
	manager, connection, err := openManagedDatabase(r.Context(), id)
	if err != nil {
		return web.DatabaseDetailData{}, err
	}
	defer manager.Close()

	const schema = "public"
	details, err := manager.ListTableDetails(r.Context(), schema)
	if err != nil {
		return web.DatabaseDetailData{}, err
	}
	if len(details) == 0 {
		return web.DatabaseDetailData{}, fmt.Errorf("no tables found in schema %q", schema)
	}
	table := r.URL.Query().Get("table")
	if table == "" {
		table = details[0].Name
	}
	data := web.DatabaseDetailData{
		BaseData: currentUser(r), Database: web.DatabaseItem{ID: id, Name: connection.name, AccessLevel: accessLevel},
		SelectedTable: table, ActiveTab: r.URL.Query().Get("tab"),
	}
	var selected bool
	for _, detail := range details {
		count, countErr := manager.CountRows(r.Context(), detail.Schema, detail.Name)
		if countErr != nil {
			slog.WarnContext(r.Context(), "db.count_rows_failed", "table", detail.Schema+"."+detail.Name, "error", countErr)
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
		return web.DatabaseDetailData{}, fmt.Errorf("table %q was not found", table)
	}
	rows, err := manager.PreviewRows(r.Context(), schema, table, 100)
	if err != nil {
		return web.DatabaseDetailData{}, err
	}
	data.DataHeaders, data.DataRows = convertRows(rows, data.Columns)
	migrations, err := manager.ListMigrations(r.Context())
	if err != nil {
		return web.DatabaseDetailData{}, err
	}
	data.Migrations = convertMigrations(migrations)
	sess, _ := r.Context().Value(auth.SessionContextKey).(*auth.Session)
	if platformDB != nil && sess != nil {
		requests, requestErr := workflow.ListRequests(platformDB, id, sess.UserID)
		if requestErr != nil {
			slog.ErrorContext(r.Context(), "migration.requests_load_failed", "database_id", id, "error", requestErr)
		} else {
			data.ChangeRequests = convertChangeRequests(requests)
		}
	}
	return data, nil
}

func renderDatabaseError(w http.ResponseWriter, r *http.Request, id int, name, tab string, err error) {
	web.RenderTemplate(w, "databasedetail.html", web.DatabaseDetailData{
		BaseData: currentUser(r), Database: web.DatabaseItem{ID: id, Name: name, AccessLevel: "view"},
		Error: err.Error(), ActiveTab: tab,
	})
}

func handleMigration(id int, operation string, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/databases/"+strconv.Itoa(id), http.StatusSeeOther)
		return
	}
	back := "/databases/" + strconv.Itoa(id)
	session, _ := r.Context().Value(auth.SessionContextKey).(*auth.Session)
	if session == nil || platformDB == nil {
		http.Error(w, "Unable to create migration request", http.StatusInternalServerError)
		return
	}
	if jobStore == nil {
		http.Redirect(w, r, back+"?error="+url.QueryEscape("migration queue is unavailable")+"#migrations", http.StatusSeeOther)
		return
	}
	action := operation
	if action == "" {
		action = "up"
	}
	mode := strings.TrimSpace(r.FormValue("mode"))
	if mode == "" {
		mode = jobs.ModePlain
	}
	logValidation := func(stage string, validationErr error, fields ...any) {
		args := []any{
			"database_id", id, "user_id", session.UserID, "action", action,
			"mode", mode, "validation_stage", stage,
		}
		args = append(args, fields...)
		if validationErr != nil {
			args = append(args, "ok", false, "error", validationErr)
			slog.WarnContext(r.Context(), "migration.validation.failed", args...)
			return
		}
		args = append(args, "ok", true)
		slog.InfoContext(r.Context(), "migration.validation.passed", args...)
	}
	if action != "up" && mode != jobs.ModePlain {
		err := errors.New("rollback and forward must use plain mode")
		logValidation("request_options", err)
		http.Redirect(w, r, back+"?error="+url.QueryEscape(err.Error())+"#migrations", http.StatusSeeOther)
		return
	}
	batchSize := 0
	if raw := strings.TrimSpace(r.FormValue("batch_size")); raw != "" {
		parsed, parseErr := strconv.Atoi(raw)
		if parseErr != nil {
			err := errors.New("batch size must be a number")
			logValidation("batch_options", err)
			http.Redirect(w, r, back+"?error="+url.QueryEscape(err.Error())+"#migrations", http.StatusSeeOther)
			return
		}
		batchSize = parsed
	}
	batchTable := strings.TrimSpace(r.FormValue("batch_table"))
	batchColumn := strings.TrimSpace(r.FormValue("batch_column"))
	var sqlUp, sqlDown, reason string
	var irreversible bool
	var targetID int64
	if action == "up" {
		sqlUp, sqlDown = r.FormValue("sql_up"), r.FormValue("sql_down")
		irreversible, reason = r.FormValue("irreversible") == "on", r.FormValue("irreversible_reason")
	} else {
		var parseErr error
		targetID, parseErr = strconv.ParseInt(r.FormValue("execution_id"), 10, 64)
		if parseErr != nil || targetID < 1 {
			err := errors.New("invalid target execution ID")
			logValidation("action_target", err)
			http.Redirect(w, r, back+"?error="+url.QueryEscape(err.Error())+"#migrations", http.StatusSeeOther)
			return
		}
		manager, _, openErr := openManagedDatabase(r.Context(), id)
		if openErr != nil {
			logValidation("action_target_database", openErr, "target_execution_id", targetID)
			http.Redirect(w, r, back+"?error="+url.QueryEscape(openErr.Error())+"#migrations", http.StatusSeeOther)
			return
		}
		target, loadErr := manager.ValidateMigrationAction(r.Context(), targetID, action)
		_ = manager.Close()
		if loadErr != nil {
			logValidation("action_eligibility", loadErr, "target_execution_id", targetID)
			http.Redirect(w, r, back+"?error="+url.QueryEscape(loadErr.Error())+"#migrations", http.StatusSeeOther)
			return
		}
		logValidation("action_eligibility", nil, "target_execution_id", targetID, "migration_version", target.Version)

		sqlUp, sqlDown = target.SQL, target.SQLDown
		irreversible, reason = target.Irreversible, nullStringValue(target.IrreversibleReason)
	}
	if mode == jobs.ModeBatch {
		if err := jobs.ValidateBatchRequest(mode, batchSize, sqlUp, batchTable, batchColumn); err != nil {
			logValidation("batch_definition", err, "sql_bytes", len(sqlUp), "batch_size", batchSize)
			http.Redirect(w, r, back+"?error="+url.QueryEscape(err.Error())+"#migrations", http.StatusSeeOther)
			return
		}
		logValidation("batch_definition", nil, "sql_bytes", len(sqlUp), "batch_size", batchSize)
	}
	request, err := workflow.CreateRequest(platformDB, id, session.UserID, session.Name, action, sqlUp, sqlDown, irreversible, reason, targetID, mode, batchSize, batchTable, batchColumn)
	if err != nil {
		logValidation("approval_request", err, "sql_bytes", len(sqlUp), "irreversible", irreversible)
		slog.WarnContext(r.Context(), "migration.request_rejected", "database_id", id, "action", action, "user_id", session.UserID, "error", err)
		http.Redirect(w, r, back+"?error="+url.QueryEscape(err.Error())+"#migrations", http.StatusSeeOther)
		return
	}
	logValidation("approval_request", nil, "request_id", request.ID, "sql_bytes", len(sqlUp),
		"irreversible", irreversible, "sql_validation", "deferred_until_execution")
	slog.InfoContext(r.Context(), "migration.requested", "request_id", request.ID, "database_id", id, "action", action, "user_id", session.UserID, "user", session.Name, "target_execution_id", targetID)
	message := fmt.Sprintf("Migration change request #%d submitted for group leader approval.", request.ID)
	http.Redirect(w, r, back+"?success="+url.QueryEscape(message)+"#migrations", http.StatusSeeOther)
}

func migrationVisibility(w http.ResponseWriter, r *http.Request) {
	databaseID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || databaseID < 1 {
		http.Error(w, "invalid database ID", http.StatusBadRequest)
		return
	}
	if !canViewDatabase(r, databaseID) {
		http.Error(w, "database not found", http.StatusNotFound)
		return
	}
	hideRollback, err := strconv.ParseBool(r.URL.Query().Get("hide_rollback"))
	if err != nil {
		http.Error(w, "hide_rollback must be true or false", http.StatusBadRequest)
		return
	}
	manager, _, err := openManagedDatabase(r.Context(), databaseID)
	if err != nil {
		slog.ErrorContext(r.Context(), "migration.visibility_load_failed", "database_id", databaseID, "error", err)
		http.Error(w, "unable to load migration history", http.StatusInternalServerError)
		return
	}
	defer manager.Close()
	migrations, err := manager.ListMigrations(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "migration.visibility_load_failed", "database_id", databaseID, "error", err)
		http.Error(w, "unable to load migration history", http.StatusInternalServerError)
		return
	}
	hiddenExecutionIDs := []int64{}
	if hideRollback {
		hiddenExecutionIDs = dbmanager.HiddenBySuccessfulRollback(migrations)
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(struct {
		HiddenExecutionIDs []int64 `json:"hidden_execution_ids"`
	}{HiddenExecutionIDs: hiddenExecutionIDs}); err != nil {
		slog.ErrorContext(r.Context(), "migration.visibility_response_failed", "database_id", databaseID, "error", err)
	}
}

func envPositiveInt(key string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil && v > 0 {
		return v
	}
	return def
}

// canViewDatabase reports whether the logged-in user has at least view access.
func canViewDatabase(r *http.Request, databaseID int) bool {
	sess, _ := r.Context().Value(auth.SessionContextKey).(*auth.Session)
	if sess == nil || platformDB == nil {
		return false
	}
	level, err := authz.GetUserDatabaseAccess(r.Context(), platformDB, sess.UserID, databaseID)
	return err == nil && level != authz.AccessNone
}

func handleMigrationDecision(databaseID int, requestID int64, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || platformDB == nil {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	session, _ := r.Context().Value(auth.SessionContextKey).(*auth.Session)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	request, err := workflow.GetRequest(platformDB, requestID)
	if err == nil && request.DatabaseID != databaseID {
		err = sql.ErrNoRows
	}
	decision := r.FormValue("decision")
	allApproved := false
	rejected := false
	queuedJobID := ""
	if err == nil {
		var ready, wasRejected bool
		ready, wasRejected, err = workflow.Respond(platformDB, requestID, session.UserID, session.Name, decision, r.FormValue("comment"))
		allApproved = ready
		rejected = wasRejected
		if err == nil {
			slog.InfoContext(r.Context(), "migration.approval_decided", "request_id", requestID, "database_id", databaseID, "decision", decision, "user_id", session.UserID, "user", session.Name)
			if rejected {
				slog.WarnContext(r.Context(), "migration.request_rejected_by_leader", "request_id", requestID, "database_id", databaseID, "approver_user_id", session.UserID)
			} else if ready {
				var queuedJob jobs.Job
				queuedJob, err = executeApprovedMigration(r.Context(), request, session.UserID, session.Name)
				if err == nil {
					queuedJobID = queuedJob.ID
				}
			}
		}
	}
	if err != nil {
		http.Redirect(w, r, "/databases/"+strconv.Itoa(databaseID)+"?error="+url.QueryEscape(err.Error())+"#migrations", http.StatusSeeOther)
		return
	}
	message := "Approval recorded. The request is waiting for the remaining group leaders."
	if rejected {
		message = "Migration change request rejected."
	} else if allApproved {
		message = "All required leaders approved the request; it was queued as job " + queuedJobID + "."
	}
	http.Redirect(w, r, "/databases/"+strconv.Itoa(databaseID)+"?success="+url.QueryEscape(message)+"#migrations", http.StatusSeeOther)
}

func convertChangeRequests(requests []workflow.Request) []web.MigrationChangeRequestItem {
	items := make([]web.MigrationChangeRequestItem, 0, len(requests))
	for _, request := range requests {
		createdAt := request.CreatedAt.Format(time.RFC3339)
		targetID := int64(0)
		if request.TargetExecutionID.Valid {
			targetID = request.TargetExecutionID.Int64
		}
		item := web.MigrationChangeRequestItem{
			ID: request.ID, Action: request.Action, RequesterName: request.RequesterName, Status: request.Status,
			TargetExecutionID: targetID, SQLUp: request.SQLUp, SQLDown: request.SQLDown,
			Irreversible: request.Irreversible, IrreversibleReason: request.IrreversibleReason,
			Mode: request.Mode, BatchSize: request.BatchSize, BatchTable: request.BatchTable, BatchColumn: request.BatchColumn,
			Error: nullStringValue(request.Error), CreatedAt: createdAt,
		}
		for _, approval := range request.Approvals {
			item.Approvals = append(item.Approvals, web.MigrationApprovalItem{
				GroupName: approval.GroupName, LeaderName: approval.LeaderName, Decision: approval.Decision,
				Comment: approval.Comment, CanRespond: approval.CanRespond,
			})
		}
		for _, event := range request.Events {
			item.Events = append(item.Events, web.MigrationWorkflowEventItem{
				Status: event.Status, ActorName: event.ActorName, Note: event.Note,
				OccurredAt: event.OccurredAt.Format(time.RFC3339),
			})
		}
		items = append(items, item)
	}
	return items
}

func executeApprovedMigration(ctx context.Context, request workflow.Request, approverID int, approverName string) (jobs.Job, error) {
	if jobStore == nil {
		return jobs.Job{}, errors.New("migration queue is unavailable")
	}
	if err := workflow.SetStatus(platformDB, request.ID, "pending_queue", approverID, approverName, "All required approvals received; enqueueing execution"); err != nil {
		return jobs.Job{}, err
	}
	if request.ResumeJobID.Valid {
		resumed, err := jobStore.ResumeApproved(ctx, request.ResumeJobID.String, request.ID, request.RequesterName)
		if err != nil {
			_ = workflow.SetStatus(platformDB, request.ID, "failed", approverID, approverName, err.Error())
			return jobs.Job{}, err
		}
		slog.InfoContext(ctx, "migration.batch_resume_queued", "request_id", request.ID,
			"job_id", resumed.ID, "database_id", resumed.DatabaseID,
			"batches", resumed.BatchesCommitted)
		return resumed, nil
	}
	sqlToRun := request.SQLUp
	version := int64(0)
	if request.Action == "rollback" {
		sqlToRun = request.SQLDown
	}
	if request.Action != "up" {
		if !request.TargetExecutionID.Valid {
			err := errors.New("migration request has no target execution ID")
			_ = workflow.SetStatus(platformDB, request.ID, "failed", approverID, approverName, err.Error())
			return jobs.Job{}, err
		}
		manager, _, err := openManagedDatabase(ctx, request.DatabaseID)
		if err != nil {
			_ = workflow.SetStatus(platformDB, request.ID, "failed", approverID, approverName, err.Error())
			return jobs.Job{}, err
		}
		var target dbmanager.Migration
		target, err = manager.ValidateMigrationAction(ctx, request.TargetExecutionID.Int64, request.Action)
		_ = manager.Close()
		if err != nil {
			_ = workflow.SetStatus(platformDB, request.ID, "failed", approverID, approverName, err.Error())
			return jobs.Job{}, err
		}
		version = target.Version
	}
	job, err := jobStore.Enqueue(ctx, jobs.EnqueueRequest{
		DatabaseID: request.DatabaseID, Kind: request.Action, Mode: request.Mode,
		BatchSize: request.BatchSize, BatchTable: request.BatchTable, BatchColumn: request.BatchColumn,
		Name: fmt.Sprintf("Migration request #%d", request.ID), SQL: sqlToRun,
		SQLDown: request.SQLDown, Irreversible: request.Irreversible, IrreversibleReason: request.IrreversibleReason,
		TargetExecutionID: request.TargetExecutionID.Int64, WorkflowRequestID: request.ID,
		Version: version, RequestedBy: request.RequesterName, RequestID: observability.RequestIDFromContext(ctx),
	})
	if err != nil {
		_ = workflow.SetStatus(platformDB, request.ID, "failed", approverID, approverName, err.Error())
		slog.ErrorContext(ctx, "migration.enqueue_failed", "request_id", request.ID, "database_id", request.DatabaseID, "action", request.Action, "error", err)
		return jobs.Job{}, err
	}
	slog.InfoContext(ctx, "migration.pending_queue", "request_id", request.ID, "job_id", job.ID,
		"database_id", request.DatabaseID, "action", request.Action, "mode", request.Mode,
		"batch_size", request.BatchSize, "batch_table", request.BatchTable, "batch_column", request.BatchColumn)
	return job, nil
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
	latestSuccess := make(map[int64]dbmanager.Migration)
	for _, migration := range migrations {
		if migration.Status == "success" {
			prior, ok := latestSuccess[migration.Version]
			if !ok || migration.ExecutionID > prior.ExecutionID {
				latestSuccess[migration.Version] = migration
			}
		}
	}
	var highestActive int64
	for version, migration := range latestSuccess {
		if migration.Action == "up" || migration.Action == "forward" {
			if version > highestActive {
				highestActive = version
			}
		}
	}
	for _, migration := range migrations {
		executedAt := ""
		if migration.ExecutedAt.Valid {
			executedAt = migration.ExecutedAt.Time.Format(time.RFC3339)
		}
		version := strconv.FormatInt(migration.Version, 10)
		latest, isLatest := latestSuccess[migration.Version]
		canRollback := isLatest && latest.ExecutionID == migration.ExecutionID && migration.Status == "success" &&
			(migration.Action == "up" || migration.Action == "forward") && migration.Version == highestActive && !migration.Irreversible
		canForward := isLatest && latest.ExecutionID == migration.ExecutionID && migration.Status == "success" &&
			migration.Action == "rollback"
		if canForward {
			for higherVersion, active := range latestSuccess {
				if higherVersion > migration.Version && (active.Action == "up" || active.Action == "forward") {
					canForward = false
					break
				}
			}
			for lowerVersion, inactive := range latestSuccess {
				if lowerVersion < migration.Version && inactive.Action == "rollback" {
					canForward = false
					break
				}
			}
		}
		targetExecutionID := int64(0)
		if migration.TargetExecutionID.Valid {
			targetExecutionID = migration.TargetExecutionID.Int64
		}
		result = append(result, web.MigrationHistoryItem{
			ExecutionID: migration.ExecutionID,
			MigrationID: migration.MigrationID.Int64, JobID: migration.JobID.String,
			RequestID: migration.RequestID.String, Revision: migration.Revision,
			Version: version, Name: "Migration " + version, ExecutedBy: migration.ExecutedBy,
			ExecutedAt: executedAt, Status: migration.Status, SQL: migration.SQL, ExecutedSQL: migration.ExecutedSQL, Action: migration.Action,
			SQLDown: migration.SQLDown, Irreversible: migration.Irreversible, IrreversibleReason: nullStringValue(migration.IrreversibleReason),
			TargetExecutionID: targetExecutionID, CanRollback: canRollback, CanForward: canForward,
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

func health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
