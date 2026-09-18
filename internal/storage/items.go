package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type ItemType string

const (
	ItemFolder ItemType = "folder"
	ItemBox    ItemType = "box"
)

type BoxType string

const (
	BoxCheck   BoxType = "check"
	BoxMeasure BoxType = "measure"
)

type Item struct {
	ID        int64
	UserID    int64
	ParentID  *int64
	Type      ItemType
	BoxType   *BoxType
	Name      string
	CreatedAt time.Time
}

func (s *Storage) CreateFolder(userID int64, parentID *int64, name string) (Item, error) {
	now := time.Now().UTC()
	res, err := s.db.Exec(
		`INSERT INTO items (user_id, parent_id, type, box_type, name, created_at) VALUES (?, ?, 'folder', NULL, ?, ?)`,
		userID, nullableID(parentID), name, now.Format(time.RFC3339),
	)
	if err != nil {
		return Item{}, fmt.Errorf("kreiranje foldera %q: %w", name, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Item{}, fmt.Errorf("čitanje id-a novog foldera: %w", err)
	}
	return Item{ID: id, UserID: userID, ParentID: parentID, Type: ItemFolder, Name: name, CreatedAt: now}, nil
}

func (s *Storage) CreateBox(userID int64, parentID *int64, boxType BoxType, name string) (Item, error) {
	now := time.Now().UTC()
	res, err := s.db.Exec(
		`INSERT INTO items (user_id, parent_id, type, box_type, name, created_at) VALUES (?, ?, 'box', ?, ?, ?)`,
		userID, nullableID(parentID), string(boxType), name, now.Format(time.RFC3339),
	)
	if err != nil {
		return Item{}, fmt.Errorf("kreiranje box-a %q: %w", name, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Item{}, fmt.Errorf("čitanje id-a novog box-a: %w", err)
	}
	bt := boxType
	return Item{ID: id, UserID: userID, ParentID: parentID, Type: ItemBox, BoxType: &bt, Name: name, CreatedAt: now}, nil
}

func (s *Storage) GetItem(id int64) (Item, error) {
	row := s.db.QueryRow(
		`SELECT id, user_id, parent_id, type, box_type, name, created_at FROM items WHERE id = ?`,
		id,
	)
	item, err := scanItem(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Item{}, fmt.Errorf("stavka %d ne postoji: %w", id, err)
		}
		return Item{}, fmt.Errorf("čitanje stavke %d: %w", id, err)
	}
	return item, nil
}

func (s *Storage) GetChildren(userID int64, parentID *int64) ([]Item, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, parent_id, type, box_type, name, created_at
		 FROM items
		 WHERE user_id = ? AND parent_id IS ?
		 ORDER BY created_at ASC, id ASC`,
		userID, nullableID(parentID),
	)
	if err != nil {
		return nil, fmt.Errorf("čitanje sadržaja (user_id=%d): %w", userID, err)
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, fmt.Errorf("skeniranje stavke: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iteracija kroz stavke: %w", err)
	}
	return items, nil
}

const descendantsCTE = `
	WITH RECURSIVE descendants(id) AS (
		SELECT id FROM items WHERE id = ?
		UNION ALL
		SELECT items.id FROM items JOIN descendants ON items.parent_id = descendants.id
	)`

func (s *Storage) CountDeletionImpact(id int64) (childItems, entryCount int, err error) {
	row := s.db.QueryRow(descendantsCTE+` SELECT COUNT(*) - 1 FROM descendants`, id)
	if err := row.Scan(&childItems); err != nil {
		return 0, 0, fmt.Errorf("brojanje potomaka stavke %d: %w", id, err)
	}

	row = s.db.QueryRow(descendantsCTE+` SELECT COUNT(*) FROM entries WHERE box_id IN (SELECT id FROM descendants)`, id)
	if err := row.Scan(&entryCount); err != nil {
		return 0, 0, fmt.Errorf("brojanje zapisa stavke %d: %w", id, err)
	}

	return childItems, entryCount, nil
}

func (s *Storage) DeleteItem(id int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("otvaranje transakcije za brisanje stavke %d: %w", id, err)
	}
	// defer tx.Rollback() will be executed at the end of the function, ensuring that if any error occurs before the transaction is committed, the transaction will be rolled back and no partial changes will be made to the database. If the transaction is successfully committed, the rollback will have no effect.
	defer tx.Rollback()

	if _, err := tx.Exec(descendantsCTE+` DELETE FROM entries WHERE box_id IN (SELECT id FROM descendants)`, id); err != nil {
		return fmt.Errorf("brisanje zapisa stavke %d: %w", id, err)
	}
	if _, err := tx.Exec(descendantsCTE+` DELETE FROM items WHERE id IN (SELECT id FROM descendants)`, id); err != nil {
		return fmt.Errorf("brisanje stavke %d: %w", id, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("potvrda brisanja stavke %d: %w", id, err)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanItem(row rowScanner) (Item, error) {
	var (
		item          Item
		parentID      sql.NullInt64
		typeText      string
		boxTypeText   sql.NullString
		createdAtText string
	)

	if err := row.Scan(&item.ID, &item.UserID, &parentID, &typeText, &boxTypeText, &item.Name, &createdAtText); err != nil {
		return Item{}, err
	}

	if parentID.Valid {
		v := parentID.Int64
		item.ParentID = &v
	}
	item.Type = ItemType(typeText)
	if boxTypeText.Valid {
		bt := BoxType(boxTypeText.String)
		item.BoxType = &bt
	}

	createdAt, err := time.Parse(time.RFC3339, createdAtText)
	if err != nil {
		return Item{}, fmt.Errorf("parsiranje created_at: %w", err)
	}
	item.CreatedAt = createdAt

	return item, nil
}
