package db

import (
	"database/sql"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Item is base data struct for incoming data.
type Item struct {
	ID       int64
	Name     string
	Path     string
	Text     string
	SaltName string
	SaltFile string
	SaltText string
	Hash     string
	Counter  int
	Created  time.Time
	Expired  time.Time
}

// FullPath returns a full path for item's file.
func (item *Item) FullPath() string {
	return filepath.Join(item.Path, item.Hash)
}

// ContentType returns string content-type for stored file.
func (item *Item) ContentType() string {
	var ext string
	i := strings.LastIndex(item.Name, ".")
	if i > -1 {
		ext = item.Name[i:]
	}
	m := mime.TypeByExtension(ext)
	if m == "" {
		return "application/octet-stream"
	}
	return m
}

// GetURL returns item's URL.
func (item *Item) GetURL(r *http.Request, secure bool) *url.URL {
	// r.URL.Scheme is blank, so use hint from settings
	scheme := "http"
	if secure {
		scheme = "https"
	}
	return &url.URL{
		Scheme: scheme,
		Host:   r.Host,
		Path:   item.Hash,
	}
}

// IsFileExists checks item's related file exists.
func (item *Item) IsFileExists() bool {
	_, err := os.Stat(item.FullPath())
	return err == nil
}

// Delete removes items from database and related file from file system.
func (item *Item) Delete(db *sql.DB) error {
	var txErr = InTransaction(db, func(tx *sql.Tx) error {
		_, err := delete(tx, item)
		return err
	})
	if txErr != nil {
		return fmt.Errorf("failed delete item by id: %w", txErr)
	}
	return deleteFiles(item)
}

// Save saves the item to database.
func (item *Item) Save(db *sql.DB) error {
	return InTransaction(db, func(tx *sql.Tx) error {
		stmt, err := tx.Prepare("INSERT INTO `storage` " +
			"(`name`, `path`, `text`, `hash`, `salt_name`, `salt_file`, `salt_text`, " +
			"`counter`, `created`, `updated`, `expired`) " +
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);")
		if err != nil {
			return fmt.Errorf("insert statement: %w", err)
		}
		result, err := tx.Stmt(stmt).Exec(item.Name, item.Path, item.Text,
			item.Hash, item.SaltName, item.SaltFile, item.SaltText,
			item.Counter, item.Created, item.Created, item.Expired)
		if err != nil {
			return fmt.Errorf("insert exec: %w", err)
		}
		item.ID, err = result.LastInsertId()
		if err != nil {
			return fmt.Errorf("insert last id: %w", err)
		}
		return nil
	})
}

// stringIDs returns comma-separated IDs of items.
func stringIDs(items []*Item) string {
	strIDs := make([]string, len(items))
	for i, item := range items {
		strIDs[i] = strconv.FormatInt(item.ID, 10)
	}
	return strings.Join(strIDs, ",")
}

// deleteFiles removes files of items.
func deleteFiles(items ...*Item) error {
	var err error
	for _, item := range items {
		err = os.RemoveAll(item.FullPath())
		if err != nil {
			return fmt.Errorf("delete file of item=%d: %w", item.ID, err)
		}
	}
	return nil
}

// Read reads an item by its hash from database.
func Read(db *sql.DB, hash string) (*Item, error) {
	stmt, err := db.Prepare("SELECT `id`, `name`, `path`, `text`, " +
		"`hash`, `salt_name`, `salt_file`, `salt_text`, `counter`, `created`, `expired` " +
		"FROM `storage` WHERE `counter`>0 AND `hash`=?;")
	if err != nil {
		return nil, fmt.Errorf("read item: %w", err)
	}
	item := &Item{}
	err = stmt.QueryRow(hash).Scan(
		&item.ID,
		&item.Name,
		&item.Path,
		&item.Text,
		&item.Hash,
		&item.SaltName,
		&item.SaltFile,
		&item.SaltText,
		&item.Counter,
		&item.Created,
		&item.Expired,
	)
	if err != nil {
		return nil, fmt.Errorf("read scan item: %w", err)
	}
	err = stmt.Close()
	if err != nil {
		return nil, fmt.Errorf("read item, close statement: %w", err)
	}
	return item, nil
}
