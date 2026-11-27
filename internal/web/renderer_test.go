package web

import (
	"html/template"
	"testing"
)

func TestEmbeddedTemplatesParse(t *testing.T) {
	if _, err := template.ParseFS(templates, "templates/*.html"); err != nil {
		t.Fatal(err)
	}
}
