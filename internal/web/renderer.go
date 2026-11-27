package web

import (
	"embed"
	"html/template"
	"net/http"
)

type BaseData struct {
	Name      string
	Role      string
	Error     string
	Success   string
	CSRFToken string
}

type DatabaseItem struct {
	ID          int
	Name        string
	AccessLevel string // "view" or "edit"
}

type DatabasesData struct {
	BaseData
	Databases []DatabaseItem
}

type LoginData struct {
	BaseData
	Username  string
	CSRFToken string
}

type DBConnectionItem struct {
	ID       int
	Database string
	Host     string
	Port     int
	User     string
	Status   string // "Connected", "Failed"
}

type GroupItem struct {
	ID          int
	Name        string
	MemberCount int
	LeaderID    int
	LeaderName  string
}

type UserItem struct {
	ID       int
	Username string
	Name     string
	Role     string // "Admin", "User"
	Groups   []string
}

type GroupPermissionItem struct {
	ID           int
	GroupID      int
	GroupName    string
	DatabaseID   int
	DatabaseName string
	AccessLevel  string // "view" or "edit"
}

type AdminData struct {
	BaseData
	ActiveTab   string
	Connections []DBConnectionItem
	Groups      []GroupItem
	Users       []UserItem
	Permissions []GroupPermissionItem
}

type ColumnItem struct {
	Name     string
	Type     string
	Nullable bool
	Default  string
	IsPK     bool
}

type ConstraintItem struct {
	Name       string
	Type       string
	Definition string
}

type IndexItem struct {
	Name    string
	Columns string
	Type    string
	Unique  bool
}

type TableSummary struct {
	Name     string
	RowCount int64
}

type MigrationHistoryItem struct {
	ExecutionID        int64
	Version            string
	MigrationID        int64
	JobID              string
	RequestID          string
	Revision           int
	Name               string
	ExecutedBy         string
	ExecutedAt         string
	Status             string
	SQL                string
	ExecutedSQL        string
	SQLDown            string
	Irreversible       bool
	IrreversibleReason string
	Action             string
	TargetExecutionID  int64
	CanRollback        bool
	CanForward         bool
}

type DatabaseDetailData struct {
	BaseData
	Database       DatabaseItem
	Tables         []TableSummary
	SelectedTable  string
	Columns        []ColumnItem
	Constraints    []ConstraintItem
	Indexes        []IndexItem
	DataHeaders    []string
	DataRows       [][]string
	Migrations     []MigrationHistoryItem
	ChangeRequests []MigrationChangeRequestItem
	ActiveTab      string
	Error          string
}

type MigrationChangeRequestItem struct {
	ID                 int64
	Action             string
	RequesterName      string
	Status             string
	TargetExecutionID  int64
	SQLUp              string
	SQLDown            string
	Irreversible       bool
	IrreversibleReason string
	Mode               string
	BatchSize          int
	BatchTable         string
	BatchColumn        string
	Error              string
	CreatedAt          string
	Approvals          []MigrationApprovalItem
	Events             []MigrationWorkflowEventItem
}

type MigrationWorkflowEventItem struct {
	Status     string
	ActorName  string
	Note       string
	OccurredAt string
}

type MigrationApprovalItem struct {
	GroupName  string
	LeaderName string
	Decision   string
	Comment    string
	CanRespond bool
}

//go:embed templates/*.html
var templates embed.FS

func RenderTemplate(w http.ResponseWriter, tmplName string, data any) {
	tmpl, err := template.ParseFS(templates, "templates/"+tmplName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
