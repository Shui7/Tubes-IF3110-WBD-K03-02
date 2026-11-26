package main

import (
	"context"
	"database/sql"
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
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/observability"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/web"
	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/workflow"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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
		}
	} else {
		slog.Error("config.missing_database_url")
		return
	}

	sessionStore := auth.NewSessionStore(platformDB, 1*time.Hour, 24*time.Hour, 5*time.Minute)
	authHandler = auth.NewAuthHandler(platformDB, sessionStore)
	authMiddleware = auth.NewAuthMiddleware(sessionStore)
	adminOnly := func(next http.Handler) http.Handler {
		return authMiddleware.RequireAuth(authMiddleware.RequireRole("Admin", next))
	}

	http.Handle("/", authMiddleware.RequireAuth(http.HandlerFunc(home)))
	http.HandleFunc("/login", authHandler.HandleLogin)
	http.HandleFunc("/logout", logout)
	http.HandleFunc("/unauthorized", unauthorizedPage)
	http.Handle("/admin", adminOnly(http.HandlerFunc(adminPage)))
	http.Handle("/admin/connections/add", adminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin.HandleAddConnection(platformDB, w, r)
	})))
	http.Handle("/admin/connections/delete", adminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin.HandleDeleteConnection(platformDB, w, r)
	})))
	http.Handle("/admin/groups/add", adminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin.HandleAddGroup(platformDB, w, r)
	})))
	http.Handle("/admin/groups/delete", adminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin.HandleDeleteGroup(platformDB, w, r)
	})))
	http.Handle("/admin/groups/leader", adminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin.HandleSetGroupLeader(platformDB, w, r)
	})))
	http.Handle("/admin/permissions/grant", adminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin.HandleGrantPermission(platformDB, w, r)
	})))
	http.Handle("/admin/permissions/delete", adminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin.HandleDeletePermission(platformDB, w, r)
	})))
	http.Handle("/admin/users/add", adminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin.HandleAddUser(platformDB, w, r)
	})))
	http.Handle("/admin/users/delete", adminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin.HandleDeleteUser(platformDB, w, r)
	})))
	http.Handle("/databases", authMiddleware.RequireAuth(http.HandlerFunc(databases)))
	http.Handle("/databases/", authMiddleware.RequireAuth(http.HandlerFunc(databases)))

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	srv := &http.Server{
		Addr:    addr,
		Handler: otelhttp.NewHandler(http.DefaultServeMux, "http.server"),
	}
	slog.InfoContext(ctx, "server.listening", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.ErrorContext(ctx, "server.failed", "error", err)
		_ = shutdownTracing(ctx) // os.Exit skips defers, so flush spans explicitly
		os.Exit(1)
	}
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
	web.RenderTemplate(w, "admin.html", data)
}

func databases(w http.ResponseWriter, r *http.Request) {
	subPath := strings.Trim(strings.TrimPrefix(r.URL.Path, "/databases"), "/")
	if subPath == "" {
		renderDatabases(w, r)
		return
	}
	parts := strings.Split(subPath, "/")
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Authorization: check user's access level for this specific database.
	sess, _ := r.Context().Value(auth.SessionContextKey).(*auth.Session)
	level, err := authz.GetUserDatabaseAccess(platformDB, sess.UserID, id)
	if err != nil || level == authz.AccessNone {
		http.Error(w, "Forbidden: you do not have access to this database", http.StatusForbidden)
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
	if len(parts) != 1 || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	renderDatabaseDetail(id, string(level), w, r)
}

func renderDatabases(w http.ResponseWriter, r *http.Request) {
	data := web.DatabasesData{BaseData: currentUser(r)}
	if platformDB == nil {
		data.Error = "Platform database is unavailable."
		web.RenderTemplate(w, "databases.html", data)
		return
	}
	sess, _ := r.Context().Value(auth.SessionContextKey).(*auth.Session)
	dbList, err := authz.GetUserDatabaseList(platformDB, sess.UserID)
	if err != nil {
		data.Error = fmt.Sprintf("Unable to load databases: %v", err)
	} else {
		for _, d := range dbList {
			data.Databases = append(data.Databases, web.DatabaseItem{
				ID: d.ConnectionID, Name: d.Name, AccessLevel: string(d.AccessLevel),
			})
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

func renderDatabaseDetail(id int, accessLevel string, w http.ResponseWriter, r *http.Request) {
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
	sess, _ := r.Context().Value(auth.SessionContextKey).(*auth.Session)
	if platformDB != nil && sess != nil {
		requests, requestErr := workflow.ListRequests(platformDB, id, sess.UserID)
		if requestErr != nil {
			slog.ErrorContext(r.Context(), "migration.requests_load_failed", "database_id", id, "error", requestErr)
		} else {
			data.ChangeRequests = convertChangeRequests(requests)
		}
	}
	web.RenderTemplate(w, "databasedetail.html", data)
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
	session, _ := r.Context().Value(auth.SessionContextKey).(*auth.Session)
	if session == nil || platformDB == nil {
		http.Error(w, "Unable to create migration request", http.StatusInternalServerError)
		return
	}
	action := operation
	if action == "" {
		action = "up"
	}
	var sqlUp, sqlDown, reason string
	var irreversible bool
	var targetID int64
	if action == "up" {
		sqlUp, sqlDown = r.FormValue("sql_up"), r.FormValue("sql_down")
		irreversible, reason = r.FormValue("irreversible") == "on", r.FormValue("irreversible_reason")
	} else {
		var parseErr error
		targetID, parseErr = strconv.ParseInt(r.FormValue("execution_id"), 10, 64)
		if parseErr != nil {
			http.Redirect(w, r, "/databases/"+strconv.Itoa(id)+"?error="+url.QueryEscape("invalid target execution ID")+"#migrations", http.StatusSeeOther)
			return
		}
		manager, _, openErr := openManagedDatabase(r.Context(), id)
		if openErr != nil {
			http.Redirect(w, r, "/databases/"+strconv.Itoa(id)+"?error="+url.QueryEscape(openErr.Error())+"#migrations", http.StatusSeeOther)
			return
		}
		target, loadErr := manager.GetMigrationExecution(r.Context(), targetID)
		_ = manager.Close()
		if loadErr != nil {
			http.Redirect(w, r, "/databases/"+strconv.Itoa(id)+"?error="+url.QueryEscape(loadErr.Error())+"#migrations", http.StatusSeeOther)
			return
		}
		sqlUp, sqlDown = target.SQL, target.SQLDown
	}
	request, err := workflow.CreateRequest(platformDB, id, session.UserID, session.Name, action, sqlUp, sqlDown, irreversible, reason, targetID)
	if err != nil {
		slog.WarnContext(r.Context(), "migration.request_rejected", "database_id", id, "action", action, "user_id", session.UserID, "error", err)
		http.Redirect(w, r, "/databases/"+strconv.Itoa(id)+"?error="+url.QueryEscape(err.Error())+"#migrations", http.StatusSeeOther)
		return
	}
	slog.InfoContext(r.Context(), "migration.requested", "request_id", request.ID, "database_id", id, "action", action, "user_id", session.UserID, "user", session.Name, "target_execution_id", targetID)
	message := fmt.Sprintf("Migration change request #%d submitted for group leader approval.", request.ID)
	http.Redirect(w, r, "/databases/"+strconv.Itoa(id)+"?success="+url.QueryEscape(message)+"#migrations", http.StatusSeeOther)
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
				err = executeApprovedMigration(r.Context(), request, session.UserID, session.Name)
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
		message = "All required leaders approved the request; execution finished."
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

func executeApprovedMigration(ctx context.Context, request workflow.Request, approverID int, approverName string) error {
	if err := workflow.SetStatus(platformDB, request.ID, "pending_queue", approverID, approverName, "All required approvals received"); err != nil {
		return err
	}
	slog.InfoContext(ctx, "migration.pending_queue", "request_id", request.ID, "database_id", request.DatabaseID, "action", request.Action)
	if err := workflow.SetStatus(platformDB, request.ID, "running", approverID, approverName, "Execution started"); err != nil {
		return err
	}
	manager, _, err := openManagedDatabase(ctx, request.DatabaseID)
	if err != nil {
		_ = workflow.SetStatus(platformDB, request.ID, "failed", approverID, approverName, err.Error())
		sqlForRequest := request.SQLUp
		if request.Action == "rollback" {
			sqlForRequest = request.SQLDown
		}
		slog.ErrorContext(ctx, "migration.execution_failed", "request_id", request.ID, "database_id", request.DatabaseID, "action", request.Action, "user_id", approverID, "target_execution_id", request.TargetExecutionID, "sql", sqlForRequest, "error", err)
		return err
	}
	defer manager.Close()

	var sqlRun string
	var version int64
	if request.Action == "up" {
		version, err = manager.NextMigrationVersion(ctx)
		sqlRun = request.SQLUp
	} else {
		var target dbmanager.Migration
		target, err = manager.GetMigrationExecution(ctx, request.TargetExecutionID.Int64)
		if err == nil {
			version = target.Version
			if request.Action == "rollback" {
				sqlRun = target.SQLDown
			} else {
				sqlRun = target.SQL
			}
		}
	}
	slog.InfoContext(ctx, "migration.execution_started", "request_id", request.ID, "database_id", request.DatabaseID, "version", version, "action", request.Action, "user_id", approverID, "user", approverName, "target_execution_id", request.TargetExecutionID, "sql", sqlRun)
	if err == nil {
		switch request.Action {
		case "up":
			err = manager.ExecuteMigrationDefinition(ctx, version, request.SQLUp, request.SQLDown, request.Irreversible, request.IrreversibleReason, approverName)
		case "rollback":
			err = manager.RollbackMigration(ctx, request.TargetExecutionID.Int64, approverName)
		case "forward":
			err = manager.ForwardMigration(ctx, request.TargetExecutionID.Int64, approverName)
		}
	}
	if err != nil {
		_ = workflow.SetStatus(platformDB, request.ID, "failed", approverID, approverName, err.Error())
		slog.ErrorContext(ctx, "migration.execution_failed", "request_id", request.ID, "database_id", request.DatabaseID, "version", version, "action", request.Action, "user_id", approverID, "user", approverName, "target_execution_id", request.TargetExecutionID, "sql", sqlRun, "error", err)
		return err
	}
	slog.InfoContext(ctx, "migration.execution_succeeded", "request_id", request.ID, "database_id", request.DatabaseID, "version", version, "action", request.Action, "user_id", approverID, "user", approverName, "target_execution_id", request.TargetExecutionID, "sql", sqlRun)
	if err := workflow.SetStatus(platformDB, request.ID, "success", approverID, approverName, "Execution completed"); err != nil {
		slog.ErrorContext(ctx, "migration.workflow_status_update_failed", "request_id", request.ID, "database_id", request.DatabaseID, "status", "success", "error", err)
		return err
	}
	return nil
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
			Version:     version, Name: "Migration " + version, ExecutedBy: migration.ExecutedBy,
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
