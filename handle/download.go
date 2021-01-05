package handle

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/z0rr0/send/cfg"
	"github.com/z0rr0/send/db"
)

// DownloadData id item's data for download page.
type DownloadData struct {
	Key       string
	CountText int
	CountFile int
}

func notFound(w http.ResponseWriter, p *Params) error {
	const tplNokName = "not_found.html"
	err := p.Settings.Tpl[cfg.NotFound].ExecuteTemplate(w, tplNokName, nil)
	if err != nil {
		return fmt.Errorf("failed execute template=%s: %w", tplNokName, err)
	}
	return nil
}

// downloadHandler generates the download page.
func downloadHandler(ctx context.Context, w http.ResponseWriter, p *Params) error {
	const tplName = "download.html"
	key := strings.Trim(p.Request.URL.Path, "/")
	_, err := uuid.Parse(key)
	if err != nil {
		return notFound(w, p)
	}
	item, err := db.Exists(ctx, p.DB, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			p.Log.Info("check item exists %s: %v", key, err)
		} else {
			p.Log.Error("check item exists %s: %v", key, err)
		}
		return notFound(w, p)
	}
	// item exists
	data := &DownloadData{Key: key, CountText: item.CountText, CountFile: item.CountFile}
	err = p.Settings.Tpl[cfg.Download].ExecuteTemplate(w, tplName, data)
	if err != nil {
		return fmt.Errorf("failed execute template=%s: %w", tplName, err)
	}
	return nil
}
