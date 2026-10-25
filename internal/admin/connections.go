package admin

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/web"
	_ "github.com/lib/pq"
)

func isPlatformDB(database string) bool {
	dbName := strings.TrimSpace(database)
	platformDB := strings.TrimSpace(os.Getenv("POSTGRES_DB"))
	if platformDB == "" {
		platformDB = "dbplatform"
	}
	return strings.EqualFold(dbName, platformDB)
}

func GetConnections(db *sql.DB) ([]web.DBConnectionItem, error) {
	rows, err := db.Query(`
		SELECT id, COALESCE(NULLIF(database_name, ''), NULLIF(name, ''), dbname, '') AS db_name, host, port, username, password 
		FROM database_connections 
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type connRow struct {
		id       int
		database string
		host     string
		port     int
		user     string
		password string
	}

	var rawConns []connRow
	for rows.Next() {
		var r connRow
		if err := rows.Scan(&r.id, &r.database, &r.host, &r.port, &r.user, &r.password); err != nil {
			return nil, err
		}
		rawConns = append(rawConns, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	connections := make([]web.DBConnectionItem, len(rawConns))
	var wg sync.WaitGroup
	for i, c := range rawConns {
		wg.Add(1)
		go func(idx int, conn connRow) {
			defer wg.Done()
			status := checkDBConnection(conn.host, conn.port, conn.database, conn.user, conn.password)
			connections[idx] = web.DBConnectionItem{
				ID:       conn.id,
				Database: conn.database,
				Host:     conn.host,
				Port:     conn.port,
				User:     conn.user,
				Status:   status,
			}
		}(i, c)
	}
	wg.Wait()

	return connections, nil
}

func checkDBConnection(host string, port int, database, user, password string) string {
	connStr := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable connect_timeout=1",
		host, port, database, user, password)
	testDB, err := sql.Open("postgres", connStr)
	if err != nil {
		return "Failed"
	}
	defer testDB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := testDB.PingContext(ctx); err != nil {
		return "Failed"
	}
	return "Connected"
}

func AddConnection(db *sql.DB, database, host string, port int, user, password string) error {
	if isPlatformDB(database) {
		return errors.New("cannot register the platform's internal database as a managed connection")
	}

	_, err := db.Exec(
		`INSERT INTO database_connections (database_name, name, dbname, host, port, username, password) 
		 VALUES ($1, $1, $1, $2, $3, $4, $5)`,
		database, host, port, user, password,
	)
	return err
}

func DeleteConnection(db *sql.DB, id int) error {
	_, err := db.Exec("DELETE FROM database_connections WHERE id = $1", id)
	return err
}

func HandleAddConnection(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin#connections", http.StatusSeeOther)
		return
	}

	database := strings.TrimSpace(r.FormValue("database"))
	host := strings.TrimSpace(r.FormValue("host"))
	portStr := strings.TrimSpace(r.FormValue("port"))
	user := strings.TrimSpace(r.FormValue("user"))
	password := r.FormValue("password")

	if isPlatformDB(database) {
		log.Printf("Blocked attempt to register platform database %q as managed connection", database)
		http.Redirect(w, r, "/admin?error=Cannot+register+the+platform's+internal+database+as+a+managed+connection#connections", http.StatusSeeOther)
		return
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 {
		port = 5432
	}

	if database != "" && host != "" && user != "" {
		if err := AddConnection(db, database, host, port, user, password); err != nil {
			log.Printf("Error adding connection: %v", err)
			http.Redirect(w, r, "/admin?error="+url.QueryEscape(err.Error())+"#connections", http.StatusSeeOther)
			return
		}
	}

	http.Redirect(w, r, "/admin#connections", http.StatusSeeOther)
}

func HandleDeleteConnection(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin#connections", http.StatusSeeOther)
		return
	}

	idStr := r.FormValue("id")
	id, err := strconv.Atoi(idStr)
	if err == nil {
		if err := DeleteConnection(db, id); err != nil {
			log.Printf("Error deleting connection: %v", err)
		}
	}

	http.Redirect(w, r, "/admin#connections", http.StatusSeeOther)
}
