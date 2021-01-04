package handle

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/z0rr0/send/db"
	"github.com/z0rr0/send/encrypt"
)

// ErrAPI is validate API error struct.
type ErrAPI struct {
	Err  string `json:"error"`
	code int
}

// Error returns a string of API error.
func (a *ErrAPI) Error() string {
	return fmt.Sprintf("%d %s", a.code, a.Err)
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

// version is API handler for version info.
func version(_ context.Context, w http.ResponseWriter, p *Params) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(p.Version)
}

// TextMeta is data struct of API response for text+meta request.
type TextMeta struct {
	Text string    `json:"text"`
	File *FileMeta `json:"file,omitempty"`
}

func validateTextAndMeta(p *Params) (string, string, error) {
	if p.Request.Method != "POST" {
		return "", "", &ErrAPI{Err: "failed HTTP method", code: http.StatusMethodNotAllowed}
	}
	password := p.Request.PostFormValue("password")
	if password == "" {
		return "", "", &ErrAPI{Err: "empty password", code: http.StatusBadRequest}
	}
	key := p.Request.PostFormValue("key")
	if key == "" {
		return "", "", &ErrAPI{Err: "empty key", code: http.StatusBadRequest}
	}
	if _, err := uuid.Parse(key); err != nil {
		return "", "", &ErrAPI{Err: "bay key", code: http.StatusBadRequest}
	}
	return password, key, nil
}

// textAndMeta is API handler to return item's text and file meta info.
func textAndMeta(ctx context.Context, w http.ResponseWriter, p *Params) error {
	var fm *FileMeta
	encoder := json.NewEncoder(w)
	password, key, err := validateTextAndMeta(p)
	if err != nil {
		e := err.(*ErrAPI)
		w.WriteHeader(e.code)
		return encoder.Encode(e)
	}
	item, err := db.Read(ctx, p.DB, key, password, nil, db.FlagText|db.FlagMeta)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrNoAttempts):
			fallthrough
		case errors.Is(err, sql.ErrNoRows):
			w.WriteHeader(http.StatusNotFound)
			return encoder.Encode(&ErrAPI{Err: "not found"})
		case errors.Is(err, encrypt.ErrSecret):
			w.WriteHeader(http.StatusBadRequest)
			return encoder.Encode(&ErrAPI{Err: "failed password or key"})
		}
		p.Log.Error("read item key=%v error: %v", key, err)
		w.WriteHeader(http.StatusInternalServerError)
		return encoder.Encode(&ErrAPI{Err: "internal error"})
	}
	if item.FileMeta != "" {
		fm, err = DecodeMeta(item.FileMeta)
		if err != nil {
			p.Log.Error("meta decode item key=%v error: %v", key, err)
			w.WriteHeader(http.StatusInternalServerError)
			return encoder.Encode(&ErrAPI{Err: "internal error"})
		}
	}
	return encoder.Encode(&TextMeta{Text: item.Text, File: fm})
}
