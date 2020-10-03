package db

import (
	"database/sql"
	"fmt"
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

// InTransaction runs method f and does commit or rollback.
func InTransaction(db *sql.DB, f func(tx *sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed transaction begin: %w", err)
	}
	err = f(tx)
	if err != nil {
		err = fmt.Errorf("failed dtransaction: %w", err)
		e := tx.Rollback()
		if e != nil {
			err = fmt.Errorf("failed rollback: %v : %w", err, e)
		}
		return err
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed transaction commit: %w", err)
	}
	return nil
}
