package storage

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type Storage struct {
	db *sql.DB
}

func Open(path string) (*Storage, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("otvaranje baze %q: %w", path, err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("povezivanje na bazu %q: %w", path, err)
	}

	s := &Storage{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Storage) migrate() error {
	const schema = `
	CREATE TABLE IF NOT EXISTS items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		parent_id INTEGER NULL REFERENCES items(id),
		type TEXT NOT NULL CHECK(type IN ('folder', 'box')),
		box_type TEXT NULL CHECK(box_type IN ('check', 'measure')),
		name TEXT NOT NULL,
		created_at TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_items_user_parent ON items(user_id, parent_id);

	CREATE TABLE IF NOT EXISTS entries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		box_id INTEGER NOT NULL REFERENCES items(id),
		value REAL NULL,
		entry_date TEXT NOT NULL,
		created_at TEXT NOT NULL,
		UNIQUE(box_id, entry_date)
	);
	CREATE INDEX IF NOT EXISTS idx_entries_box_id ON entries(box_id);
	`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("migracija seme baze: %w", err)
	}
	return nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

// nil becomes NULL in SQL, non-nil becomes the value itself. This is useful for optional foreign keys (parent_id) and optional values (value).
func nullableID(id *int64) any {
	if id == nil {
		return nil
	}
	return *id
}
