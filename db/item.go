package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/z0rr0/send/encrypt"
	"github.com/z0rr0/send/logging"
)

// DecryptFlag is a type for decryption flags.
type DecryptFlag uint8

// decryption flags
const (
	FlagText DecryptFlag = 1 << iota
	FlagMeta
	FlagFile
)

// Item is base data struct for incoming data.
type Item struct {
	ID        int64
	Key       string
	Text      string
	FileMeta  string
	FilePath  string
	CountText int
	CountFile int
	HashText  string
	HashFile  string
	HashMeta  string
	SaltText  string
	SaltFile  string
	SaltMeta  string
	Created   time.Time
	Updated   time.Time
	Expired   time.Time
	// without saving to db
	AutoPassword bool
	Storage      string
	ErrLogger    *logging.Log
}

func (item *Item) encryptText(secret string, e error) error {
	if e != nil {
		return e
	}
	if item.Text == "" {
		return nil
	}
	m, err := encrypt.Text(secret, item.Text)
	if err != nil {
		return err
	}
	item.Text = m.Value
	item.HashText = m.Hash
	item.SaltText = m.Salt
	return nil
}

func (item *Item) decryptText(secret string, e error) error {
	if e != nil {
		return e
	}
	if item.Text == "" {
		// nothing to decrypt
		return nil
	}
	m := &encrypt.Msg{Salt: item.SaltText, Value: item.Text, Hash: item.HashText}
	plainText, err := encrypt.DecryptText(secret, m)
	if err != nil {
		return err
	}
	item.Text = plainText
	return nil
}

func (item *Item) encryptFileMeta(secret string, e error) error {
	if e != nil {
		return e
	}
	if item.FileMeta == "" {
		return nil
	}
	m, err := encrypt.Text(secret, item.FileMeta)
	if err != nil {
		return err
	}
	item.FileMeta = m.Value
	item.HashMeta = m.Hash
	item.SaltMeta = m.Salt
	return nil
}

func (item *Item) decryptFileMeta(secret string, e error) error {
	if e != nil {
		return e
	}
	if item.FileMeta == "" {
		return nil
	}
	m := &encrypt.Msg{Salt: item.SaltMeta, Value: item.FileMeta, Hash: item.HashMeta}
	plainText, err := encrypt.DecryptText(secret, m)
	if err != nil {
		return err
	}
	item.FileMeta = plainText
	return nil
}

func (item *Item) encryptFile(secret string, src io.Reader, e error) error {
	if e != nil {
		return e
	}
	if item.FileMeta == "" {
		return nil
	}
	if src == nil {
		return errors.New("not file for encryption")
	}
	m, err := encrypt.File(secret, src, item.Storage, item.Key)
	if err != nil {
		return err
	}
	item.FilePath = m.Value
	item.HashFile = m.Hash
	item.SaltFile = m.Salt
	return nil
}

func (item *Item) decryptFile(secret string, dst io.Writer, e error) error {
	if e != nil {
		return e
	}
	if item.FileMeta == "" {
		return nil
	}
	m := &encrypt.Msg{Salt: item.SaltFile, Hash: item.HashFile}
	return encrypt.DecryptFile(secret, m, dst)
}

// Encrypt updates item's fields by encrypted values.
func (item *Item) Encrypt(secret string, src io.Reader) error {
	var err error
	err = item.encryptText(secret, err)
	err = item.encryptFileMeta(secret, err)
	return item.encryptFile(secret, src, err)
}

// Decrypt updates item's fields by decrypted values.
func (item *Item) Decrypt(secret string, dst io.Writer, flags DecryptFlag) error {
	var err error
	if flags&FlagText != 0 {
		err = item.decryptText(secret, err)
	}
	if flags&FlagMeta != 0 {
		err = item.decryptFileMeta(secret, err)
	}
	if flags&FlagFile != 0 {
		err = item.decryptFile(secret, dst, err)
	}
	return err
}

// ContentType returns string content-type for stored file.
func (item *Item) ContentType() string {
	const defaultContent = "application/octet-stream"
	var ext string
	i := strings.LastIndex(item.FileMeta, ".")
	if i > -1 {
		ext = item.FileMeta[i:]
	}
	m := mime.TypeByExtension(ext)
	if m == "" {
		return defaultContent
	}
	return m
}

// GetURL returns item's URL.
func (item *Item) GetURL(r *http.Request, secure bool) *url.URL {
	// r.URL.Scheme is blank, so use hint from settings
	var scheme = "http"
	if secure {
		scheme = "https"
	}
	return &url.URL{
		Scheme: scheme,
		Host:   r.Host,
		Path:   item.Key,
	}
}

// String returns a string representation of Item.
func (item *Item) String() string {
	return fmt.Sprintf("Item{%s}", item.Key)
}

// IsFileExists checks item's related file exists.
func (item *Item) IsFileExists() bool {
	_, err := os.Stat(item.FilePath)
	return err == nil
}

// Delete removes items from database and related file from file system.
func (item *Item) Delete(ctx context.Context, db *sql.DB) error {
	var txErr = InTransaction(ctx, db, func(tx *sql.Tx) error {
		// ignore number of affected rows
		// the item can be deleted before by GC
		_, err := deleteItems(ctx, tx, item)
		return err
	})
	if txErr != nil {
		return fmt.Errorf("failed deleteItems item by id: %w", txErr)
	}
	return deleteFiles(item)
}

// Save saves the item to thd db database.
func (item *Item) Save(ctx context.Context, db *sql.DB) error {
	const insertSQL = "INSERT INTO `storage` " +
		"(`key`,`text`,`file_meta`,`file_path`,`count_text`,`count_file`," +
		"`hash_text`,`hash_meta`,`hash_file`,`salt_text`,`salt_meta`,`salt_file`," +
		"`created`,`updated`,`expired`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?);"
	return InTransaction(ctx, db, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, insertSQL)
		if err != nil {
			return fmt.Errorf("insert statement: %w", err)
		}
		result, err := tx.StmtContext(ctx, stmt).ExecContext(ctx,
			item.Key, item.Text, item.FileMeta, item.FilePath, item.CountText, item.CountFile,
			item.HashText, item.HashMeta, item.HashFile, item.SaltText, item.SaltMeta, item.SaltFile,
			item.Created, item.Created, item.Expired,
		)
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

// IsActive returns true if item still have available counters.
func (item *Item) IsActive() bool {
	return item.CountText > 0 || item.CountFile > 0
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
	for _, item := range items {
		if item.FilePath == "" {
			continue
		}
		err := os.Remove(item.FilePath)
		if err != nil {
			return fmt.Errorf("deleteItems file of item=%d: %w", item.ID, err)
		}
	}
	return nil
}

// Read reads an item by its key from the database.
func Read(ctx context.Context, db *sql.DB, key string) (*Item, error) {
	const readSQL = "SELECT `id`,`key`,`text`,`file_meta`,`file_path`,`count_text`,`count_file`," +
		"`hash_text`,`hash_meta`,`hash_file`,`salt_text`,`salt_meta`,`salt_file`," +
		"`created`,`updated`,`expired` " +
		"FROM `storage` " +
		"WHERE `key`=? AND `expired`<=? ((`count_text`>0) OR (`count_file`>0));"
	stmt, err := db.PrepareContext(ctx, readSQL)
	if err != nil {
		return nil, fmt.Errorf("read item: %w", err)
	}
	item := &Item{}
	err = stmt.QueryRowContext(ctx, key, time.Now().UTC()).Scan(
		&item.ID, &item.Key, &item.Text, &item.FileMeta, &item.FilePath, &item.CountText, &item.CountFile,
		&item.HashText, &item.HashMeta, &item.HashFile, &item.HashFile, &item.SaltText, &item.SaltMeta, &item.SaltFile,
		&item.Created, &item.Updated, &item.Expired,
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
