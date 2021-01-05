package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/z0rr0/send/logging"
)

// InTransaction runs method `f` inside the database transaction and does commit or rollback.
func InTransaction(ctx context.Context, db *sql.DB, f func(tx *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed transaction begin: %w", err)
	}
	err = f(tx)
	if err != nil {
		err = fmt.Errorf("rollback transaction: %w", err)
		e := tx.Rollback()
		if e != nil {
			err = fmt.Errorf("failed rollback: %v: %w", err, e)
		}
		return err
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed transaction commit: %w", err)
	}
	return nil
}

// expired returns already expired items for now timestamp or it they have not active counters.
func expired(ctx context.Context, tx *sql.Tx) ([]*Item, error) {
	const expiredSQL = "SELECT `id`, `file_path` " +
		"FROM `storage` " +
		"WHERE `expired`<? OR (`count_text`<1 AND `count_file`<1) " +
		"ORDER BY `id`;"
	var items []*Item
	stmt, err := tx.PrepareContext(ctx, expiredSQL)
	if err != nil {
		return nil, fmt.Errorf("prepare select expired query: %w", err)
	}
	rows, err := tx.StmtContext(ctx, stmt).QueryContext(ctx, time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("exec select expired query: %w", err)
	}
	for rows.Next() {
		item := &Item{}
		err = rows.Scan(&item.ID, &item.FilePath)
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

// deleteItems removes items by their identifiers.
func deleteItems(ctx context.Context, tx *sql.Tx, items ...*Item) (int64, error) {
	// prepare sql query directly with arguments to exclude multi-items parsing
	var deleteQuery = fmt.Sprintf("DELETE FROM `storage` WHERE `id` IN (%s);", stringIDs(items))
	stmt, err := tx.PrepareContext(ctx, deleteQuery)
	if err != nil {
		return 0, fmt.Errorf("prepare deleteItems query: %w", err)
	}
	result, err := stmt.ExecContext(ctx)
	if err != nil {
		return 0, fmt.Errorf("can not exec deleteItems transaction: %w", err)
	}
	return result.RowsAffected()
}

// deleteByDateOrCounters removes expired items.
func deleteByDateOrCounters(ctx context.Context, db *sql.DB) (int64, error) {
	var n int64
	var txErr = InTransaction(ctx, db, func(tx *sql.Tx) error {
		items, err := expired(ctx, tx)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		n, err = deleteItems(ctx, tx, items...)
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
func GCMonitor(ch <-chan Item, shutdown, done chan struct{}, db *sql.DB, tickT, dbT time.Duration, l *logging.Log) {
	var (
		cancel context.CancelFunc
		ctx    context.Context
		ticker = time.NewTicker(tickT)
	)
	defer func() {
		ticker.Stop()
		close(done)
		l.Info("gc monitor stopped")
	}()
	l.Info("GC monitor is running, period=%v", tickT)
	for {
		select {
		case item := <-ch:
			ctx, cancel = context.WithTimeout(context.Background(), dbT)
			if err := item.Delete(ctx, db); err != nil {
				l.Error("failed deleteItems %s: %v", item.String(), err)
			} else {
				l.Info("deleted %s", item.String())
			}
			cancel()
		case <-ticker.C:
			ctx, cancel = context.WithTimeout(context.Background(), dbT)
			if n, err := deleteByDateOrCounters(ctx, db); err != nil {
				l.Error("failed deleteItems item(s) by date: %v", err)
			} else {
				if n > 0 {
					l.Info("deleted %v expired item(s)", n)
				}
			}
			cancel()
		case <-shutdown:
			return
		}
	}
}
