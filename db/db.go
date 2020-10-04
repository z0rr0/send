package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
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

// deleteByIDs removes items by their identifiers.
func deleteByIDs(tx *sql.Tx, ids ...int64) (int64, error) {
	stmt, err := tx.Prepare("DELETE FROM `storage` WHERE `id` IN (?);")
	if err != nil {
		return 0, fmt.Errorf("can not prepare delete transaction: %w", err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			logErr.Printf("failed close stmt: %v\n", err)
		}
	}()
	strIDs := make([]string, len(ids))
	for i, v := range ids {
		strIDs[i] = strconv.FormatInt(v, 10)
	}
	result, err := stmt.Exec(strings.Join(strIDs, ","))
	if err != nil {
		return 0, fmt.Errorf("can not exec delete transaction: %w", err)
	}
	return result.RowsAffected()
}

// deleteByDate removes expired items.
func deleteByDate(db *sql.DB) (int64, error) {
	return 0, nil
}

// GCMonitor is garbage collection monitoring to delete expired by date or counter items.
func GCMonitor(ch <-chan Item, closed chan struct{}, db *sql.DB, period time.Duration) {
	tc := time.Tick(period)
	logInfo.Printf("GC monitor is running, period=%v\n", period)
	for {
		select {
		case item := <-ch:
			if err := item.Delete(db); err != nil {
				logErr.Println(err)
			} else {
				logInfo.Printf("deleted item=%v\n", item.ID)
			}
		case <-tc:
			if n, err := deleteByDate(db); err != nil {
				logErr.Println(err)
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
