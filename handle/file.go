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

// fileHandler is handler to return content of the file.
func fileHandler(ctx context.Context, w http.ResponseWriter, p *Params) (int, error) {
	ajax := p.Request.PostFormValue("ajax") == "true"
	password, key, e := validatePassKey(p)
	if e != nil {
		e.ajax = ajax
		p.Log.Info("password/key validation failed: %v", e.Err)
		return downloadErrHandler(w, p, e)
	}
	// read/decrement fileMeta+file, but decrypt only fileMeta data due to dst=nil
	item, err := db.Read(ctx, p.DB, key, password, nil, db.FlagMeta|db.FlagFile)
	if err != nil {
		e = &ErrItem{Err: "internal error", Code: http.StatusInternalServerError, ajax: ajax}
		switch {
		case errors.Is(err, db.ErrNoAttempts):
			fallthrough
		case errors.Is(err, sql.ErrNoRows):
			e.Code, e.Err = http.StatusNotFound, "not found"
			return downloadErrHandler(w, p, e)
		case errors.Is(err, encrypt.ErrSecret):
			e.Code, e.Err, e.Key = http.StatusBadRequest, "failed secret", key
			return downloadErrHandler(w, p, e)
		}
		p.Log.Error("read item file key=%v error: %v", key, err)
		return downloadErrHandler(w, p, e)
	}
	defer item.CheckCounts(p.DelItem)
	// password is already valid and item was decremented for file and fileMeta
	if item.FileMeta == "" {
		return downloadErrHandler(w, p, &ErrItem{Err: "no content", Code: http.StatusNoContent, ajax: ajax})
	}
	fileMeta, err := DecodeMeta(item.FileMeta)
	if err != nil {
		p.Log.Error("fileMeta decode item file key=%v error: %v", key, err)
		return downloadErrHandler(w, p, &ErrItem{Err: "internal error", Code: http.StatusInternalServerError, ajax: ajax})
	}
	w.Header().Set("Content-Type", fileMeta.ResponseContentType())
	w.Header().Set("Content-Disposition", fileMeta.ResponseContentDisposition())
	w.Header().Set("Content-Length", fileMeta.ResponseContentLength())
	err = item.Decrypt(password, w, db.FlagFile, nil)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}
