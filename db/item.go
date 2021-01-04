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

var (
	// ErrDecrement is an error when decrement was failed.
	ErrDecrement = errors.New("can not decrement item")
	// ErrNoAttempts is an error when there are no attempts to read some data
	ErrNoAttempts = errors.New("no more attempts")

	// all decryption flags
	flagSlice = [3]DecryptFlag{FlagText, FlagMeta, FlagMeta}
)

// Item is base data struct for incoming data.
type Item struct {
	ID        int64
	Key       string
	Text      string
	FileMeta  string
	FilePath  string
	CountText int
	CountMeta int
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
func (item *Item) Decrypt(secret string, dst io.Writer, flags DecryptFlag, err error) error {
	if err != nil {
		return err
	}
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
// TODO: delete me
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
		"(`key`,`text`,`file_meta`,`file_path`,`count_text`,`count_meta`,`count_file`," +
		"`hash_text`,`hash_meta`,`hash_file`,`salt_text`,`salt_meta`,`salt_file`," +
		"`created`,`updated`,`expired`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?);"
	return InTransaction(ctx, db, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, insertSQL)
		if err != nil {
			return fmt.Errorf("insert statement: %w", err)
		}
		result, err := tx.StmtContext(ctx, stmt).ExecContext(ctx,
			item.Key, item.Text, item.FileMeta, item.FilePath, item.CountText, item.CountMeta, item.CountFile,
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

// read loads an unexpired Item from database by the key.
func (item *Item) read(ctx context.Context, tx *sql.Tx, key string) error {
	const readSQL = "SELECT `id`,`key`,`text`,`file_meta`,`file_path`," +
		"`count_text`,`count_meta`,`count_file`," +
		"`hash_text`,`hash_meta`,`hash_file`," +
		"`salt_text`,`salt_meta`,`salt_file`," +
		"`created`,`updated`,`expired` " +
		"FROM `storage` " +
		"WHERE `key`=? AND `expired`>=? AND ((`count_text`>0) OR (`count_file`>0));"
	stmt, err := tx.PrepareContext(ctx, readSQL)
	if err != nil {
		return fmt.Errorf("read item statement: %w", err)
	}
	return stmt.QueryRowContext(ctx, key, time.Now().UTC()).Scan(
		&item.ID, &item.Key, &item.Text, &item.FileMeta, &item.FilePath,
		&item.CountText, &item.CountMeta, &item.CountFile,
		&item.HashText, &item.HashMeta, &item.HashFile,
		&item.SaltText, &item.SaltMeta, &item.SaltFile,
		&item.Created, &item.Updated, &item.Expired,
	)
}

// validate checks that there are attempts to read requested data.
func (item *Item) validate(flags DecryptFlag, err error) error {
	if err != nil {
		return err
	}
	noText := (flags&FlagText != 0) && (item.CountText < 1)
	noMeta := (flags&FlagMeta != 0) && (item.CountMeta < 1)
	noFile := (flags&FlagFile != 0) && (item.CountFile < 1)
	if noText || noMeta || noFile {
		return ErrNoAttempts
	}
	return nil
}

// decrement updates item in the database, decrements its counters.
func (item *Item) decrement(ctx context.Context, tx *sql.Tx, flags DecryptFlag, err error) error {
	if err != nil {
		return err
	}
	const updateSQL = "UPDATE `storage` " +
		"SET `count_text`=`count_text`-?, `count_meta`=`count_meta`-?, `count_file`=`count_file`-?, `updated`=? " +
		"WHERE `id`=?;"
	counters := make(map[DecryptFlag]int)
	stmt, err := tx.PrepareContext(ctx, updateSQL)
	if err != nil {
		return fmt.Errorf("update statement: %w", err)
	}
	for _, flagValue := range flagSlice {
		if flags&flagValue != 0 {
			counters[flagValue] = 1
		}
	}
	result, err := tx.StmtContext(ctx, stmt).ExecContext(
		ctx, counters[FlagText], counters[FlagMeta], counters[FlagFile], time.Now().UTC(), item.ID,
	)
	if err != nil {
		return fmt.Errorf("exec update item: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check updated rows after decrement: %w", err)
	}
	if n != 1 {
		return ErrDecrement
	}
	return nil
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
// It also decrypts it and decrements counters.
func Read(ctx context.Context, db *sql.DB, key, password string, dst io.Writer, flags DecryptFlag) (*Item, error) {
	item := &Item{}
	err := InTransaction(ctx, db, func(tx *sql.Tx) error {
		// move 1st found error `e` through all methods
		// to don't check `if e != nil` after every call
		e := item.read(ctx, tx, key)
		e = item.validate(flags, e)
		e = item.Decrypt(password, dst, flags, e)
		return item.decrement(ctx, tx, flags, e)
	})
	if err != nil {
		return nil, err
	}
	return item, nil
}
