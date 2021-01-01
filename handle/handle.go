package handle

// Package handle contains HTTP web/api handling methods.

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/z0rr0/send/cfg"
	"github.com/z0rr0/send/db"
	"github.com/z0rr0/send/logging"
)

// Params is a struct with common handling arguments.
type Params struct {
	Log      *logging.Log
	Settings *cfg.Settings
	Request  *http.Request
	Version  *Version
	DelItem  chan<- db.Item
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

// Main is a common HTTP handler.
func Main(w io.Writer, p *Params) error {
	var handler func(io.Writer, *Params) error

	switch p.Request.URL.Path {
	case "/":
		handler = index
	case "/api/version":
		handler = version
	default:
		// download by hash
		handler = index
	}
	return handler(w, p)
}

func index(w io.Writer, p *Params) error {
	data := struct {
		MaxSize int
	}{MaxSize: p.Settings.Size}

	err := p.Settings.Tpl.ExecuteTemplate(w, "index.html", data)
	if err != nil {
		return fmt.Errorf("failed execute template=index.html: %w", err)
	}
	return nil
}
