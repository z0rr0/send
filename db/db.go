package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"
)

var (
	// internal loggers
	logErr  = log.New(os.Stderr, "ERROR [db]: ", log.Ldate|log.Ltime|log.Lshortfile)
	logInfo = log.New(os.Stdout, "INFO [db]: ", log.Ldate|log.Ltime)
)

// InTransaction runs method f inside a database transaction and does commit or rollback.
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

// expired returns already expired items for now timestamp.
func expired(tx *sql.Tx) ([]*Item, error) {
	var items []*Item
	stmt, err := tx.Prepare("SELECT `id`, `path`, `hash` FROM `storage` WHERE `expired`<?;")
	if err != nil {
		return nil, fmt.Errorf("prepare select expired query: %w", err)
	}
	rows, err := tx.Stmt(stmt).Query(time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("exec select expired query: %w", err)
	}
	for rows.Next() {
		item := &Item{}
		err = rows.Scan(&item.ID, &item.Path, &item.Hash)
		if err != nil {
			return nil, fmt.Errorf("next select expired query: %w", err)
		}
		items = append(items, item)
	}
	err = rows.Close()
	if err != nil {
		return nil, fmt.Errorf("close rows expired query: %w", err)
	}
	return items, nil
}

// deleteItems removes items by their identifiers.
func deleteItems(tx *sql.Tx, items ...*Item) (int64, error) {
	// statement will be closed when the transaction has been committed or rolled back
	stmt, err := tx.Prepare("DELETE FROM `storage` WHERE `id` IN (?);")
	if err != nil {
		return 0, fmt.Errorf("can not prepare deleteItems transaction: %w", err)
	}
	result, err := tx.Stmt(stmt).Exec(stringIDs(items))
	if err != nil {
		return 0, fmt.Errorf("can not exec deleteItems transaction: %w", err)
	}
	return result.RowsAffected()
}

// deleteByDate removes expired items.
func deleteByDate(db *sql.DB) (int64, error) {
	var n int64
	var txErr = InTransaction(db, func(tx *sql.Tx) error {
		items, err := expired(tx)
		if err != nil {
			return err
		}
		n, err = deleteItems(tx, items...)
		if err != nil {
			return err
		}
		return deleteFiles(items...)
	})
	if txErr != nil {
		return 0, fmt.Errorf("failed deleteItems item by date: %w", txErr)
	}
	return n, nil
}

// GCMonitor is garbage collection monitoring to delete expired by date or counter items.
func GCMonitor(ch <-chan Item, closed chan struct{}, db *sql.DB, period time.Duration) {
	tc := time.Tick(period)
	logInfo.Printf("GC monitor is running, period=%v\n", period)
	for {
		select {
		case item := <-ch:
			if err := item.Delete(db); err != nil {
				logErr.Printf("failed deleteItems item: %v\n", err)
			} else {
				logInfo.Printf("deleted item=%v\n", item.ID)
			}
		case <-tc:
			if n, err := deleteByDate(db); err != nil {
				logErr.Printf("failed deleteItems items by date: %v\n", err)
			} else {
				if n > 0 {
					logInfo.Printf("deleted %v expired items\n", n)
				}
			}
		case <-closed:
			logInfo.Println("gc monitor stopped")
			return
		}
	}
}
