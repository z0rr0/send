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

// errPassKey is validate API error struct.
type errPassKey struct {
	Err  string `json:"error"`
	code int
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

// version is API handler for app info.
func version(_ context.Context, w http.ResponseWriter, p *Params) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(p.Version)
}

// TextMeta is data struct of API response for text+meta request.
type TextMeta struct {
	Text string    `json:"text"`
	File *FileMeta `json:"file,omitempty"`
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

// textAPI is API handler to return item's text and file meta info.
func textAPI(ctx context.Context, w http.ResponseWriter, p *Params) error {
	var fileMeta *FileMeta
	encoder := json.NewEncoder(w)
	password, key, e := validatePassKey(p)
	if e != nil {
		w.WriteHeader(e.code)
		return encoder.Encode(e)
	}
	item, err := db.Read(ctx, p.DB, key, password, nil, db.FlagText|db.FlagMeta, p.DelItem)
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
	if item.FileMeta != "" {
		fileMeta, err = DecodeMeta(item.FileMeta)
		if err != nil {
			p.Log.Error("fileMeta decode item key=%v error: %v", key, err)
			return err
		}
	}
	return encoder.Encode(&TextMeta{Text: item.Text, File: fileMeta})
}

// fileAPI is API handler to return content file data.
func fileAPI(ctx context.Context, w http.ResponseWriter, p *Params) error {
	password, key, e := validatePassKey(p)
	if e != nil {
		p.Log.Info("password/key validation failed: %v", e.Err)
		w.WriteHeader(e.code)
		return nil
	}
	// read/decrement fileMeta+file, but decrypt only fileMeta data due to dst=nil
	item, err := db.Read(ctx, p.DB, key, password, nil, db.FlagMeta|db.FlagFile, p.DelItem)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrNoAttempts):
			fallthrough
		case errors.Is(err, sql.ErrNoRows):
			w.WriteHeader(http.StatusNotFound)
			return nil
		case errors.Is(err, encrypt.ErrSecret):
			w.WriteHeader(http.StatusBadRequest)
			return nil
		}
		p.Log.Error("read item file key=%v error: %v", key, err)
		return err
	}
	// password is already valid and item was decremented for file and fileMeta
	if item.FileMeta != "" {
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
	fileMeta, err := DecodeMeta(item.FileMeta)
	if err != nil {
		p.Log.Error("fileMeta decode item file key=%v error: %v", key, err)
		return err
	}
	w.Header().Set("Content-Type", fileMeta.ResponseContentType())
	w.Header().Set("Content-Disposition", fileMeta.ResponseContentDisposition())
	w.Header().Set("Content-Length", fileMeta.ResponseContentLength())
	return item.Decrypt(password, w, db.FlagFile, nil)
}
