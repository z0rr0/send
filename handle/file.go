package handle

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/z0rr0/send/db"
	"github.com/z0rr0/send/encrypt"
)

// FileMeta is base file data.
type FileMeta struct {
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
}

// Encode converts file metadata to a json string.
func (f *FileMeta) Encode() (string, error) {
	b, err := json.Marshal(f)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ResponseContentType returns HTTP content-type.
func (f *FileMeta) ResponseContentType() string {
	if f.ContentType == "" {
		return "application/octet-stream"
	}
	return f.ContentType
}

// ResponseContentDisposition returns HTTP content-disposition.
func (f *FileMeta) ResponseContentDisposition() string {
	return fmt.Sprintf("attachment; filename=\"%s\"", f.Name)
}

// ResponseContentLength returns HTTP content-length.
func (f *FileMeta) ResponseContentLength() string {
	return strconv.FormatInt(f.Size, 10)
}

// DecodeMeta returns a parsed from json string file metadata.
func DecodeMeta(fileMeta string) (*FileMeta, error) {
	f := &FileMeta{}
	err := json.Unmarshal([]byte(fileMeta), f)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func failedDownload(w http.ResponseWriter, code int, msg string) error {
	w.WriteHeader(code)
	_, err := fmt.Fprint(w, msg)
	return err
}

// fileHandler is handler to return content of the file.
func fileHandler(ctx context.Context, w http.ResponseWriter, p *Params) error {
	password, key, e := validatePassKey(p)
	if e != nil {
		p.Log.Info("password/key validation failed: %v", e.Err)
		return failedDownload(w, e.code, "secret validation failed")
	}
	// read/decrement fileMeta+file, but decrypt only fileMeta data due to dst=nil
	item, err := db.Read(ctx, p.DB, key, password, nil, db.FlagMeta|db.FlagFile)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrNoAttempts):
			fallthrough
		case errors.Is(err, sql.ErrNoRows):
			return failedDownload(w, http.StatusNotFound, "not found")
		case errors.Is(err, encrypt.ErrSecret):
			return failedDownload(w, http.StatusBadRequest, "secret validation failed")
		}
		p.Log.Error("read item file key=%v error: %v", key, err)
		return failedDownload(w, http.StatusInternalServerError, "internal error")
	}
	defer item.CheckCounts(p.DelItem)
	// password is already valid and item was decremented for file and fileMeta
	if item.FileMeta == "" {
		return failedDownload(w, http.StatusNoContent, "no content")
	}
	fileMeta, err := DecodeMeta(item.FileMeta)
	if err != nil {
		p.Log.Error("fileMeta decode item file key=%v error: %v", key, err)
		return failedDownload(w, http.StatusInternalServerError, "internal error")
	}
	w.Header().Set("Content-Type", fileMeta.ResponseContentType())
	w.Header().Set("Content-Disposition", fileMeta.ResponseContentDisposition())
	w.Header().Set("Content-Length", fileMeta.ResponseContentLength())
	return item.Decrypt(password, w, db.FlagFile, nil)
}
