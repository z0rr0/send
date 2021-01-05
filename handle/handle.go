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

// errPassKey is validate API error struct.
type errPassKey struct {
	Err  string `json:"error"`
	code int
}

func validatePassKey(p *Params) (string, string, *errPassKey) {
	if p.Request.Method != "POST" {
		return "", "", &errPassKey{Err: "failed HTTP method", code: http.StatusMethodNotAllowed}
	}
	password := p.Request.PostFormValue("password")
	if password == "" {
		return "", "", &errPassKey{Err: "empty password", code: http.StatusBadRequest}
	}
	key := p.Request.PostFormValue("key")
	if key == "" {
		return "", "", &errPassKey{Err: "empty key", code: http.StatusBadRequest}
	}
	if _, err := uuid.Parse(key); err != nil {
		return "", "", &errPassKey{Err: "bad key", code: http.StatusBadRequest}
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
	const tplName = "index.html"
	data := &IndexData{MaxSize: p.Settings.Size}
	err := p.Settings.Tpl[cfg.Index].ExecuteTemplate(w, tplName, data)
	if err != nil {
		return fmt.Errorf("failed execute template=%s: %w", tplName, err)
	}
	return nil
}
