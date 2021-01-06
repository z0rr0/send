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
	CountText bool
	CountFile bool
}

func notFound(w http.ResponseWriter, p *Params, ei *ErrItem) error {
	if ei == nil {
		ei = &ErrItem{Err: "Not found", code: 404}
	}
	w.WriteHeader(ei.code)
	err := p.Settings.Tpl[cfg.ErrorTpl].ExecuteTemplate(w, cfg.ErrorTpl, ei)
	if err != nil {
		return fmt.Errorf("failed execute template=%s: %w", cfg.ErrorTpl, err)
	}
	return nil
}

// downloadHandler generates the download page.
func downloadHandler(ctx context.Context, w http.ResponseWriter, p *Params) error {
	key := strings.Trim(p.Request.URL.Path, "/ ")
	_, err := uuid.Parse(key)
	if err != nil {
		return notFound(w, p, nil)
	}
	item, err := db.Exists(ctx, p.DB, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			p.Log.Info("check item exists %s: %v", key, err)
			return notFound(w, p, nil)
		}
		p.Log.Error("check item exists %s: %v", key, err)
		return notFound(w, p, &ErrItem{Err: "Internal error", code: 500})
	}
	data := &DownloadData{
		Key:       key,
		CountText: item.CountText > 0,
		CountFile: item.CountFile > 0,
	}
	err = p.Settings.Tpl[cfg.DownloadTpl].ExecuteTemplate(w, cfg.DownloadTpl, data)
	if err != nil {
		return fmt.Errorf("failed execute template=%s: %w", cfg.DownloadTpl, err)
	}
	return nil
}
