package web

import (
	"embed"
	"html/template"
	"net/http"
)

type BaseData struct {
	Name    string
	Role    string
	Error   string
	Success string
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
	Username string
}

type DBConnectionItem struct {
	ID       int
	Database string
	Host     string
	Port     int
	User     string
	Status   string // "Connected", "Failed", "Untested"
}

type GroupItem struct {
	ID          int
	Name        string
	MemberCount int
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
