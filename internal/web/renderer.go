package web

import (
	"embed"
	"html/template"
	"net/http"
)

type BaseData struct {
	Name      string `json:"name"`
	Role      string `json:"role"`
	Error     string `json:"error"`
	Success   string `json:"success"`
	CSRFToken string `json:"csrfToken"`
}

type DatabaseItem struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	AccessLevel string `json:"accessLevel"` // "view" or "edit"
}

type DatabasesData struct {
	BaseData
	Databases []DatabaseItem `json:"databases"`
}

type LoginData struct {
	BaseData
	Username string `json:"username"`
}

type DBConnectionItem struct {
	ID       int    `json:"id"`
	Database string `json:"database"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Status   string `json:"status"` // "Connected", "Failed"
}

type GroupItem struct {
	ID             int
	Name           string
	MemberCount    int
	LeaderID       int
	LeaderName     string
	LeaderIsMember bool
	Members        []GroupMemberItem
}

type GroupMemberItem struct {
	ID       int
	Username string
	Name     string
}

type UserItem struct {
	ID       int      `json:"id"`
	Username string   `json:"username"`
	Name     string   `json:"name"`
	Role     string   `json:"role"` // "Admin", "User"
	Groups   []string `json:"groups"`
}

type GroupPermissionItem struct {
	ID           int    `json:"id"`
	GroupID      int    `json:"groupId"`
	GroupName    string `json:"groupName"`
	DatabaseID   int    `json:"databaseId"`
	DatabaseName string `json:"databaseName"`
	AccessLevel  string `json:"accessLevel"` // "view" or "edit"
}

type AdminData struct {
	BaseData
	ActiveTab   string                `json:"activeTab"`
	Connections []DBConnectionItem    `json:"connections"`
	Groups      []GroupItem           `json:"groups"`
	Users       []UserItem            `json:"users"`
	Permissions []GroupPermissionItem `json:"permissions"`
}

type ColumnItem struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Default  string `json:"default"`
	IsPK     bool   `json:"isPk"`
}

type ConstraintItem struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Definition string `json:"definition"`
}

type IndexItem struct {
	Name    string `json:"name"`
	Columns string `json:"columns"`
	Type    string `json:"type"`
	Unique  bool   `json:"unique"`
}

type TableSummary struct {
	Name     string `json:"name"`
	RowCount int64  `json:"rowCount"`
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
