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
	ID      int64
	Name    string
	Path    string
	Text    string
	Salt    string
	Hash    string
	Counter int
	Created time.Time
	Expired time.Time
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
		_, err := deleteByIDs(tx, item)
		return err
	})
	if txErr != nil {
		return fmt.Errorf("failed delete item by id: %w", txErr)
	}
	return deleteFiles(item)
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
