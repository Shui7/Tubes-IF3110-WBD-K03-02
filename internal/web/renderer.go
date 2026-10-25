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
