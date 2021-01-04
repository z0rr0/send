package handle

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/z0rr0/send/db"
	"github.com/z0rr0/send/encrypt"
)

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

// versionHandler is API handler for app info.
func versionHandler(_ context.Context, w http.ResponseWriter, p *Params) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(p.Version)
}

// TextMeta is data struct of API response for text+meta request.
type TextMeta struct {
	Text string    `json:"text"`
	File *FileMeta `json:"file,omitempty"`
}

// textAPIHandler is API handler to return item's text and file meta info.
func textAPIHandler(ctx context.Context, w http.ResponseWriter, p *Params) error {
	var fileMeta *FileMeta
	encoder := json.NewEncoder(w)
	password, key, e := validatePassKey(p)
	if e != nil {
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
			return encoder.Encode(&errPassKey{Err: "not found"})
		case errors.Is(err, encrypt.ErrSecret):
			w.WriteHeader(http.StatusBadRequest)
			return encoder.Encode(&errPassKey{Err: "failed password or key"})
		}
		p.Log.Error("read item key=%v error: %v", key, err)
		return err
	}
	defer item.CheckCounts(p.DelItem)
	if item.FileMeta != "" {
		fileMeta, err = DecodeMeta(item.FileMeta)
		if err != nil {
			p.Log.Error("fileMeta decode item key=%v error: %v", key, err)
			return err
		}
	}
	return encoder.Encode(&TextMeta{Text: item.Text, File: fileMeta})
}
