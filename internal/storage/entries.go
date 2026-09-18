package storage

import (
	"database/sql"
	"fmt"
	"time"
)

const DateFormat = "2006-01-02"

type Entry struct {
	ID        int64
	BoxID     int64
	Value     *float64
	EntryDate string // format DateFormat
	CreatedAt time.Time
}

// if the measurement for the current date already exists, it will be updated with the new value. If it does not exist, a new entry will be created. This is done in a single SQL statement using "INSERT ... ON CONFLICT ... DO UPDATE", which ensures atomicity and avoids race conditions.
func (s *Storage) UpsertMeasureEntry(boxID int64, date string, value float64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`INSERT INTO entries (box_id, value, entry_date, created_at) VALUES (?, ?, ?, ?)
		 ON CONFLICT(box_id, entry_date) DO UPDATE SET value = excluded.value`,
		boxID, value, date, now,
	)
	if err != nil {
		return fmt.Errorf("upis zapisa za box %d: %w", boxID, err)
	}
	return nil
}

func (s *Storage) MarkCheckDone(boxID int64, date string) (alreadyMarked bool, err error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(
		`INSERT OR IGNORE INTO entries (box_id, value, entry_date, created_at) VALUES (?, NULL, ?, ?)`,
		boxID, date, now,
	)
	if err != nil {
		return false, fmt.Errorf("upis check zapisa za box %d: %w", boxID, err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("provera upisanog reda: %w", err)
	}
	return rows == 0, nil
}

func (s *Storage) GetEntries(boxID int64) ([]Entry, error) {
	rows, err := s.db.Query(
		`SELECT id, box_id, value, entry_date, created_at FROM entries WHERE box_id = ? ORDER BY entry_date ASC`,
		boxID,
	)
	if err != nil {
		return nil, fmt.Errorf("čitanje zapisa za box %d: %w", boxID, err)
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var (
			e             Entry
			value         sql.NullFloat64
			createdAtText string
		)
		if err := rows.Scan(&e.ID, &e.BoxID, &value, &e.EntryDate, &createdAtText); err != nil {
			return nil, fmt.Errorf("skeniranje reda: %w", err)
		}
		if value.Valid {
			v := value.Float64
			e.Value = &v
		}
		createdAt, err := time.Parse(time.RFC3339, createdAtText)
		if err != nil {
			return nil, fmt.Errorf("parsiranje created_at: %w", err)
		}
		e.CreatedAt = createdAt
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iteracija kroz zapise: %w", err)
	}
	return entries, nil
}
