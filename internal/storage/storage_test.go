package storage

import (
	"path/filepath"
	"testing"
)

// openTestStorage kreira svežu SQLite bazu u privremenom direktorijumu za
// svaki test. t.TempDir() vraća direktorijum koji Go test runner sam čisti
// posle testa - nema potrebe za ručnim "defer os.RemoveAll(...)".
func openTestStorage(t *testing.T) *Storage {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestCreateFolderAndBox(t *testing.T) {
	s := openTestStorage(t)
	const userID = int64(1)

	folder, err := s.CreateFolder(userID, nil, "Trening")
	if err != nil {
		t.Fatalf("CreateFolder() error = %v", err)
	}
	if folder.Type != ItemFolder || folder.ParentID != nil {
		t.Fatalf("CreateFolder() = %+v, očekivan root folder", folder)
	}

	box, err := s.CreateBox(userID, &folder.ID, BoxMeasure, "Plank")
	if err != nil {
		t.Fatalf("CreateBox() error = %v", err)
	}
	if box.Type != ItemBox || box.BoxType == nil || *box.BoxType != BoxMeasure {
		t.Fatalf("CreateBox() = %+v, očekivan measure box", box)
	}
	if box.ParentID == nil || *box.ParentID != folder.ID {
		t.Fatalf("CreateBox() parentID = %v, očekivano %d", box.ParentID, folder.ID)
	}
}

func TestGetChildrenRootVsFolder(t *testing.T) {
	s := openTestStorage(t)
	const userID = int64(1)

	folder, err := s.CreateFolder(userID, nil, "Trening")
	if err != nil {
		t.Fatalf("CreateFolder() error = %v", err)
	}
	if _, err := s.CreateBox(userID, nil, BoxCheck, "Vitamini"); err != nil {
		t.Fatalf("CreateBox(root) error = %v", err)
	}
	if _, err := s.CreateBox(userID, &folder.ID, BoxMeasure, "Plank"); err != nil {
		t.Fatalf("CreateBox(folder) error = %v", err)
	}

	// GetChildren(nil) treba da vrati root stavke (i folder i root box), ne
	// stavke unutar foldera - proverava "parent_id IS ?" trik za NULL.
	root, err := s.GetChildren(userID, nil)
	if err != nil {
		t.Fatalf("GetChildren(root) error = %v", err)
	}
	if len(root) != 2 {
		t.Fatalf("GetChildren(root) vratio %d stavki, očekivano 2", len(root))
	}

	inFolder, err := s.GetChildren(userID, &folder.ID)
	if err != nil {
		t.Fatalf("GetChildren(folder) error = %v", err)
	}
	if len(inFolder) != 1 || inFolder[0].Name != "Plank" {
		t.Fatalf("GetChildren(folder) = %+v, očekivan samo Plank", inFolder)
	}
}

func TestNestedFolders(t *testing.T) {
	s := openTestStorage(t)
	const userID = int64(1)

	outer, err := s.CreateFolder(userID, nil, "Zdravlje")
	if err != nil {
		t.Fatalf("CreateFolder(root) error = %v", err)
	}
	inner, err := s.CreateFolder(userID, &outer.ID, "Trening")
	if err != nil {
		t.Fatalf("CreateFolder(folder-in-folder) error = %v", err)
	}
	if inner.ParentID == nil || *inner.ParentID != outer.ID {
		t.Fatalf("CreateFolder() parentID = %v, očekivano %d", inner.ParentID, outer.ID)
	}
	if _, err := s.CreateBox(userID, &inner.ID, BoxMeasure, "Plank"); err != nil {
		t.Fatalf("CreateBox(nested folder) error = %v", err)
	}

	root, err := s.GetChildren(userID, nil)
	if err != nil {
		t.Fatalf("GetChildren(root) error = %v", err)
	}
	if len(root) != 1 || root[0].ID != outer.ID {
		t.Fatalf("GetChildren(root) = %+v, očekivan samo Zdravlje", root)
	}

	inOuter, err := s.GetChildren(userID, &outer.ID)
	if err != nil {
		t.Fatalf("GetChildren(outer) error = %v", err)
	}
	if len(inOuter) != 1 || inOuter[0].ID != inner.ID {
		t.Fatalf("GetChildren(outer) = %+v, očekivan samo Trening", inOuter)
	}

	inInner, err := s.GetChildren(userID, &inner.ID)
	if err != nil {
		t.Fatalf("GetChildren(inner) error = %v", err)
	}
	if len(inInner) != 1 || inInner[0].Name != "Plank" {
		t.Fatalf("GetChildren(inner) = %+v, očekivan samo Plank", inInner)
	}
}

func TestDeleteItemCascadesThroughNestedFolders(t *testing.T) {
	s := openTestStorage(t)
	const userID = int64(1)

	outer, err := s.CreateFolder(userID, nil, "Zdravlje")
	if err != nil {
		t.Fatalf("CreateFolder(root) error = %v", err)
	}
	inner, err := s.CreateFolder(userID, &outer.ID, "Trening")
	if err != nil {
		t.Fatalf("CreateFolder(nested) error = %v", err)
	}
	box, err := s.CreateBox(userID, &inner.ID, BoxMeasure, "Plank")
	if err != nil {
		t.Fatalf("CreateBox() error = %v", err)
	}
	if err := s.UpsertMeasureEntry(box.ID, "2026-09-16", 42); err != nil {
		t.Fatalf("UpsertMeasureEntry() error = %v", err)
	}

	childItems, entryCount, err := s.CountDeletionImpact(outer.ID)
	if err != nil {
		t.Fatalf("CountDeletionImpact() error = %v", err)
	}
	if childItems != 2 || entryCount != 1 {
		t.Fatalf("CountDeletionImpact() = (%d, %d), očekivano (2, 1)", childItems, entryCount)
	}

	if err := s.DeleteItem(outer.ID); err != nil {
		t.Fatalf("DeleteItem() error = %v", err)
	}

	if _, err := s.GetItem(outer.ID); err == nil {
		t.Fatal("GetItem(outer) uspeo posle brisanja, očekivana greška")
	}
	if _, err := s.GetItem(inner.ID); err == nil {
		t.Fatal("GetItem(inner) uspeo posle brisanja - ugnježdeni folder nije obrisan")
	}
	if _, err := s.GetItem(box.ID); err == nil {
		t.Fatal("GetItem(box) uspeo posle brisanja - box unutar obrisanog foldera nije obrisan")
	}

	entries, err := s.GetEntries(box.ID)
	if err != nil {
		t.Fatalf("GetEntries() error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("GetEntries() = %+v, očekivano 0 zapisa posle brisanja box-a", entries)
	}
}

func TestDeleteItemOnlyAffectsItsOwnSubtree(t *testing.T) {
	s := openTestStorage(t)
	const userID = int64(1)

	folder, err := s.CreateFolder(userID, nil, "Trening")
	if err != nil {
		t.Fatalf("CreateFolder() error = %v", err)
	}
	box, err := s.CreateBox(userID, &folder.ID, BoxCheck, "Vitamini")
	if err != nil {
		t.Fatalf("CreateBox() error = %v", err)
	}

	if err := s.DeleteItem(box.ID); err != nil {
		t.Fatalf("DeleteItem(box) error = %v", err)
	}

	if _, err := s.GetItem(folder.ID); err != nil {
		t.Fatalf("GetItem(folder) error = %v posle brisanja box-a - folder ne bi trebalo da je pogođen", err)
	}
	if _, err := s.GetItem(box.ID); err == nil {
		t.Fatal("GetItem(box) uspeo posle brisanja, očekivana greška")
	}
}

func TestMarkCheckDoneNoDuplicate(t *testing.T) {
	s := openTestStorage(t)
	box, err := s.CreateBox(1, nil, BoxCheck, "Vitamini")
	if err != nil {
		t.Fatalf("CreateBox() error = %v", err)
	}

	alreadyMarked, err := s.MarkCheckDone(box.ID, "2026-09-16")
	if err != nil {
		t.Fatalf("MarkCheckDone() #1 error = %v", err)
	}
	if alreadyMarked {
		t.Fatal("MarkCheckDone() prvi put vratio alreadyMarked = true")
	}

	alreadyMarked, err = s.MarkCheckDone(box.ID, "2026-09-16")
	if err != nil {
		t.Fatalf("MarkCheckDone() #2 error = %v", err)
	}
	if !alreadyMarked {
		t.Fatal("MarkCheckDone() drugi put vratio alreadyMarked = false, očekivan duplikat")
	}

	entries, err := s.GetEntries(box.ID)
	if err != nil {
		t.Fatalf("GetEntries() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("GetEntries() vratio %d zapisa, očekivan 1 (bez duplikata)", len(entries))
	}
	if entries[0].Value != nil {
		t.Fatalf("GetEntries()[0].Value = %v, očekivano nil za check box", *entries[0].Value)
	}
}

func TestUpsertMeasureEntryUpdatesSameDay(t *testing.T) {
	s := openTestStorage(t)
	box, err := s.CreateBox(1, nil, BoxMeasure, "Kilaza")
	if err != nil {
		t.Fatalf("CreateBox() error = %v", err)
	}

	if err := s.UpsertMeasureEntry(box.ID, "2026-09-16", 82.5); err != nil {
		t.Fatalf("UpsertMeasureEntry() #1 error = %v", err)
	}
	if err := s.UpsertMeasureEntry(box.ID, "2026-09-16", 83.0); err != nil {
		t.Fatalf("UpsertMeasureEntry() #2 error = %v", err)
	}

	entries, err := s.GetEntries(box.ID)
	if err != nil {
		t.Fatalf("GetEntries() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("GetEntries() vratio %d zapisa, očekivan 1 (upsert, ne duplikat)", len(entries))
	}
	if entries[0].Value == nil || *entries[0].Value != 83.0 {
		t.Fatalf("GetEntries()[0].Value = %v, očekivano 83.0 (ažurirana vrednost)", entries[0].Value)
	}
}
