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
func versionHandler(_ context.Context, w http.ResponseWriter, p *Params) (int, error) {
	encoder := json.NewEncoder(w)
	err := encoder.Encode(p.Version)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

// TextMeta is data struct of API response for text+meta request.
type TextMeta struct {
	Text string    `json:"text"`
	File *FileMeta `json:"file,omitempty"`
}

// textAPIHandler is API handler to return item's text and file meta info.
func textAPIHandler(ctx context.Context, w http.ResponseWriter, p *Params) (int, error) {
	var fileMeta *FileMeta
	encoder := json.NewEncoder(w)
	password, key, e := validatePassKey(p)
	if e != nil {
		w.WriteHeader(e.Code)
		if err := encoder.Encode(e); err != nil {
			return http.StatusInternalServerError, err
		}
		return e.Code, nil
	}
	item, err := db.Read(ctx, p.DB, key, password, nil, db.FlagText|db.FlagMeta)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrNoAttempts):
			fallthrough
		case errors.Is(err, sql.ErrNoRows):
			w.WriteHeader(http.StatusNotFound)
			err = encoder.Encode(&ErrItem{Err: "not found"})
			if err != nil {
				return http.StatusInternalServerError, err
			}
			return http.StatusNotFound, nil
		case errors.Is(err, encrypt.ErrSecret):
			w.WriteHeader(http.StatusBadRequest)
			err = encoder.Encode(&ErrItem{Err: "failed password or key"})
			if err != nil {
				return http.StatusInternalServerError, err
			}
			return http.StatusBadRequest, nil
		}
		p.Log.Error("read item key=%v error: %v", key, err)
		return http.StatusInternalServerError, err
	}
	defer item.CheckCounts(p.DelItem)
	if item.FileMeta != "" {
		fileMeta, err = DecodeMeta(item.FileMeta)
		if err != nil {
			p.Log.Error("fileMeta decode item key=%v error: %v", key, err)
			return http.StatusInternalServerError, err
		}
	}
	err = encoder.Encode(&TextMeta{Text: item.Text, File: fileMeta})
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}
