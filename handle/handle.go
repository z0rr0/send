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

type handlerType func(context.Context, http.ResponseWriter, *Params) (int, error)

// Params is a struct with common handling arguments.
// Except Request field, it is read-only struct.
type Params struct {
	Log      *logging.Log
	DB       *sql.DB
	Settings *cfg.Settings
	Request  *http.Request
	Version  *Version
	DelItem  chan<- db.Item
	Storage  *cfg.Storage
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
	Key  string `json:"-"`
	Code int    `json:"-"`
	ajax bool
}

// Error returns a string representation of the error.
func (e *ErrItem) Error() string {
	return fmt.Sprintf("%d %s", e.Code, e.Err)
}

// HasKey returns true if the item has a filled key field.
func (e *ErrItem) HasKey() bool {
	return e.Key != ""
}

func validatePassKey(p *Params) (string, string, *ErrItem) {
	if p.Request.Method != "POST" {
		return "", "", &ErrItem{Err: "failed HTTP method", Code: http.StatusMethodNotAllowed}
	}
	password := p.Request.PostFormValue("password")
	if password == "" {
		return "", "", &ErrItem{Err: "empty password", Code: http.StatusBadRequest}
	}
	key := p.Request.PostFormValue("key")
	if key == "" {
		return "", "", &ErrItem{Err: "empty key", Code: http.StatusBadRequest}
	}
	if _, err := uuid.Parse(key); err != nil {
		return "", "", &ErrItem{Err: "bad key", Code: http.StatusBadRequest}
	}
	return password, key, nil
}

// downloadErrHandler is a handler method to return some error page/message.
func downloadErrHandler(w http.ResponseWriter, p *Params, ei *ErrItem) (int, error) {
	var err error
	if ei == nil {
		ei = &ErrItem{Err: "Not found", Code: 404}
	}
	w.WriteHeader(ei.Code)
	if ei.ajax {
		_, err = fmt.Fprint(w, ei.Err)
		if err != nil {
			return http.StatusInternalServerError, err
		}
		return ei.Code, nil
	}
	err = p.Settings.Tpl[cfg.ErrorTpl].ExecuteTemplate(w, cfg.ErrorTpl, ei)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed execute template=%s: %w", cfg.ErrorTpl, err)
	}
	return ei.Code, nil
}

// Main is a common HTTP handler.
func Main(ctx context.Context, w http.ResponseWriter, p *Params) int {
	var handlers = map[string]handlerType{
		"/":            indexHandler,
		"/upload":      uploadHandler,
		"/file":        fileHandler,
		"/api/version": versionHandler,
		"/api/text":    textAPIHandler,
		"/api/upload":  uploadAPIHandler,
		// "/UUID":     downloadHandler,
	}
	handler, ok := handlers[p.Request.URL.Path]
	if !ok {
		// download by UUID, 32 hex: 8-4-4-4-12
		handler = downloadHandler
	}
	code, err := handler(ctx, w, p)
	if err != nil {
		p.Log.Error("error: %v", err)
		return http.StatusInternalServerError
	}
	return code
}

// indexHandler is a title web page.
func indexHandler(_ context.Context, w http.ResponseWriter, p *Params) (int, error) {
	data := &IndexData{MaxSize: p.Settings.Size}
	err := p.Settings.Tpl[cfg.IndexTpl].ExecuteTemplate(w, cfg.IndexTpl, data)
	if err != nil {
		return 0, fmt.Errorf("failed execute template=%s: %w", cfg.IndexTpl, err)
	}
	return http.StatusOK, nil
}
