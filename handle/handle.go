package handle

// Package handle contains HTTP web/api handling methods.

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"

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

// Version is application details info.
type Version struct {
	Version     string `json:"version"`
	Revision    string `json:"revision"`
	Build       string `json:"build"`
	Environment string `json:"environment"`
}

// String returns a string representation of Version struct.
func (v *Version) String() string {
	return fmt.Sprintf("Version: %s\nRevision: %s\nBuild date: %s\nGo version: %s",
		v.Version, v.Revision, v.Build, v.Environment,
	)
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

// Main is a common HTTP handler.
func Main(ctx context.Context, w http.ResponseWriter, p *Params) error {
	var handlers = map[string]handlerType{
		"/":            index,
		"/upload":      upload,
		"/api/version": version,
	}
	handler, ok := handlers[p.Request.URL.Path]
	if !ok {
		// download by hash
		// 32 hex: 8-4-4-4-12
		handler = index
	}
	return handler(ctx, w, p)
}

// index is a title web page.
func index(_ context.Context, w http.ResponseWriter, p *Params) error {
	const tplName = "index.html"
	data := &IndexData{MaxSize: p.Settings.Size}
	err := p.Settings.Tpl[cfg.Index].ExecuteTemplate(w, tplName, data)
	if err != nil {
		return fmt.Errorf("failed execute template=%s: %w", tplName, err)
	}
	return nil
}
