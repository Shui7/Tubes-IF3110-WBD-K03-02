package main

import (
	"log"
	"net/http"

	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/web"
)

func main() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("/app/static"))))

	http.HandleFunc("/", home)
	http.HandleFunc("/databases/", databases)

	log.Println("Listening at http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func currentUser() web.BaseData {
	// placeholder
	return web.BaseData{
		Name: "John User",
		Role: "User",
	}
}

func home(w http.ResponseWriter, r *http.Request) {
	web.RenderTemplate(w, "home.html", currentUser())
}

func databases(w http.ResponseWriter, r *http.Request) {
	// placeholder
	web.RenderTemplate(w, "databases.html", web.DatabasesData{
		BaseData: currentUser(),
		Databases: []web.DatabaseItem{
			{ID: 1, Name: "sales_db", AccessLevel: "edit"},
			{ID: 2, Name: "reports_db", AccessLevel: "view"},
		},
	})
}
