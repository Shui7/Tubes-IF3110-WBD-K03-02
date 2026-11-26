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
	ID          int    `json:"id"`
	Name        string `json:"name"`
	MemberCount int    `json:"memberCount"`
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
	Version    string `json:"version"`
	Name       string `json:"name"`
	ExecutedBy string `json:"executedBy"`
	ExecutedAt string `json:"executedAt"`
	Status     string `json:"status"`
	SQL        string `json:"sql"`
}

type DatabaseDetailData struct {
	BaseData
	Database      DatabaseItem           `json:"database"`
	Tables        []TableSummary         `json:"tables"`
	SelectedTable string                 `json:"selectedTable"`
	Columns       []ColumnItem           `json:"columns"`
	Constraints   []ConstraintItem       `json:"constraints"`
	Indexes       []IndexItem            `json:"indexes"`
	DataHeaders   []string               `json:"dataHeaders"`
	DataRows      [][]string             `json:"dataRows"`
	Migrations    []MigrationHistoryItem `json:"migrations"`
	ActiveTab     string                 `json:"activeTab"`
	Error         string                 `json:"error"`
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
