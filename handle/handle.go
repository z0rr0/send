package handle

// Package handle contains HTTP web/api handling methods.

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/z0rr0/send/cfg"
	"github.com/z0rr0/send/db"
	"github.com/z0rr0/send/logging"
)

type handlerType func(context.Context, http.ResponseWriter, *Params) error

// Params is a struct with common handling arguments.
// Except Request field, it is read-only struct.
type Params struct {
	Log      *logging.Log
	DB       *sql.DB
	Settings *cfg.Settings
	Request  *http.Request
	Version  *Version
	DelItem  chan<- db.Item
	Storage  string
	Secure   bool
}

// IsAPI returns true if params are for API requests.
func (p *Params) IsAPI() bool {
	return strings.HasPrefix(p.Request.URL.Path, "/api")
}

// IndexData is index page data.
type IndexData struct {
	MaxSize int
	Error   string
}

// HasError returns true if there is an error message.
func (iData *IndexData) HasError() bool {
	return iData.Error != ""
}

// ErrItem is validate API error struct.
type ErrItem struct {
	Err  string `json:"error"`
	code int
}

// Error returns a string representation of the error.
func (e *ErrItem) Error() string {
	return fmt.Sprintf("%d %s", e.code, e.Err)
}

func validatePassKey(p *Params) (string, string, *ErrItem) {
	if p.Request.Method != "POST" {
		return "", "", &ErrItem{Err: "failed HTTP method", code: http.StatusMethodNotAllowed}
	}
	password := p.Request.PostFormValue("password")
	if password == "" {
		return "", "", &ErrItem{Err: "empty password", code: http.StatusBadRequest}
	}
	key := p.Request.PostFormValue("key")
	if key == "" {
		return "", "", &ErrItem{Err: "empty key", code: http.StatusBadRequest}
	}
	if _, err := uuid.Parse(key); err != nil {
		return "", "", &ErrItem{Err: "bad key", code: http.StatusBadRequest}
	}
	return password, key, nil
}

// Main is a common HTTP handler.
func Main(ctx context.Context, w http.ResponseWriter, p *Params) error {
	var handlers = map[string]handlerType{
		"/":            indexHandler,
		"/upload":      uploadHandler,
		"/file":        fileHandler,
		"/api/version": versionHandler,
		"/api/text":    textAPIHandler,
	}
	handler, ok := handlers[p.Request.URL.Path]
	if !ok {
		// download by UUID, 32 hex: 8-4-4-4-12
		handler = downloadHandler
	}
	return handler(ctx, w, p)
}

// indexHandler is a title web page.
func indexHandler(_ context.Context, w http.ResponseWriter, p *Params) error {
	data := &IndexData{MaxSize: p.Settings.Size}
	err := p.Settings.Tpl[cfg.IndexTpl].ExecuteTemplate(w, cfg.IndexTpl, data)
	if err != nil {
		return fmt.Errorf("failed execute template=%s: %w", cfg.IndexTpl, err)
	}
	return nil
}
