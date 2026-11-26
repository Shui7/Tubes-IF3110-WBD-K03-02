package managed

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/Shui7/Tubes-IF3110-WBD-K03-02/internal/dbmanager"
)

type entry struct {
	dsn string
	mgr *dbmanager.Manager
}

// Registry caches a dbmanager.Manager per database ID. If the stored connection
// settings change, the old pool is closed and a new one is opened.
type Registry struct {
	platformDB *sql.DB
	mu         sync.Mutex
	entries    map[int]entry
}

func NewRegistry(platformDB *sql.DB) *Registry {
	return &Registry{platformDB: platformDB, entries: make(map[int]entry)}
}

func quote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "\\'") + "'"
}

func (r *Registry) dsn(ctx context.Context, id int) (string, error) {
	var host, database, username, password string
	var port int
	err := r.platformDB.QueryRowContext(ctx, `
		SELECT host, port,
		       COALESCE(NULLIF(database_name, ''), NULLIF(name, ''), dbname, ''),
		       username, password
		FROM database_connections WHERE id = $1`, id).Scan(&host, &port, &database, &username, &password)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("database %d was not found", id)
	}
	if err != nil {
		return "", fmt.Errorf("load database %d: %w", id, err)
	}
	return fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		quote(host), port, quote(database), quote(username), quote(password)), nil
}

// Get returns the manager for a database. The caller must not close it.
func (r *Registry) Get(ctx context.Context, id int) (*dbmanager.Manager, error) {
	dsn, err := r.dsn(ctx, id)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if e, ok := r.entries[id]; ok {
		if e.dsn == dsn {
			return e.mgr, nil
		}
		_ = e.mgr.Close()
		delete(r.entries, id)
	}
	mgr, err := dbmanager.Open(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect to database %d: %w", id, err)
	}
	r.entries[id] = entry{dsn: dsn, mgr: mgr}
	return mgr, nil
}

func (r *Registry) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, e := range r.entries {
		_ = e.mgr.Close()
		delete(r.entries, id)
	}
}
