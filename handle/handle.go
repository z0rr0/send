package handle

// Package handle contains HTTP web/api handling methods.

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/z0rr0/send/cfg"
	"github.com/z0rr0/send/logging"
	"github.com/z0rr0/send/tpl"
)

// Main is a common HTTP handler.
func Main(ctx context.Context, w io.Writer, r *http.Request) error {
	logger, err := logging.Get(ctx)
	if err != nil {
		return err
	}
	logger.Info("get url=%v", r.URL.Path)
	t, err := tpl.GetByName(ctx, tpl.Index)
	if err != nil {
		return fmt.Errorf("get template=%s: %w", tpl.Index, err)
	}
	settings, err := cfg.GetSettings(ctx)
	if err != nil {
		return fmt.Errorf("get setings: %w", err)
	}
	data := struct {
		MaxSize int
	}{
		settings.Size,
	}
	err = t.Execute(w, data)
	if err != nil {
		return fmt.Errorf("execute template=%s: %w", tpl.Index, err)
	}
	return nil

	//switch r.URL.Path {
	//case "/":
	//case "/version":
	//default:
	//}
}
