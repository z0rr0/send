package handle

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/z0rr0/send/db"
	"github.com/z0rr0/send/encrypt"
)

// defaultPasswordBytes is bytes for hex password generation
const defaultPasswordBytes = 10

// validateInt checks that field is in the range [1; max].
func validateInt(name, value string, max int) (int, error) {
	const min = 1
	v, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("validation value of %s: %w", name, err)
	}
	if (v < min) || (v > max) {
		return 0, fmt.Errorf("validation of %s, value=%d is not in [%d; %d]", name, v, min, max)
	}
	return v, nil
}

// failedUpload returns index.html page with error message.
func failedUpload(w http.ResponseWriter, status int, data *IndexData, p *Params) error {
	const tplName = "index.html"
	w.WriteHeader(status)
	err := p.Settings.Tpl.ExecuteTemplate(w, tplName, data)
	if err != nil {
		return fmt.Errorf("failed execute template=%s: %w", tplName, err)
	}
	return nil
}

func validateUpload(w http.ResponseWriter, p *Params) (*db.Item, error) {
	var fileName string
	data := &IndexData{MaxSize: p.Settings.Size}
	if p.Request.Method != "POST" {
		data.Error = "failed HTTP method"
		return nil, failedUpload(w, http.StatusMethodNotAllowed, data, p)
	}
	// file
	f, h, err := p.Request.FormFile("file")
	if err != nil {
		if !errors.Is(err, http.ErrMissingFile) {
			data.Error = "failed file upload"
			return nil, failedUpload(w, http.StatusBadRequest, data, p)
		}
		// ErrMissingFile will be checked later with text-field
	} else {
		fileName = h.Filename
	}
	defer func() {
		if e := p.Request.Body.Close(); e != nil {
			p.Log.Error("close request body: %v", e)
		}
		if fileName != "" {
			if e := f.Close(); e != nil {
				p.Log.Error("close incoming file: %v", e)
			}
		}
	}()
	// text
	text := p.Request.PostFormValue("text")
	if fileName == "" && text == "" {
		data.Error = "empty text and file fields"
		return nil, failedUpload(w, http.StatusBadRequest, data, p)
	}
	// ttl
	ttl, err := validateInt("TTL", p.Request.PostFormValue("ttl"), p.Settings.TTL)
	if err != nil {
		data.Error = "incorrect TTL"
		return nil, failedUpload(w, http.StatusBadRequest, data, p)
	}
	// times
	times, err := validateInt("times", p.Request.PostFormValue("times"), p.Settings.Times)
	if err != nil {
		data.Error = "incorrect times"
		return nil, failedUpload(w, http.StatusBadRequest, data, p)
	}
	// password
	password := p.Request.PostFormValue("password")
	if password == "" {
		// auto generation
		pwd, e := encrypt.Random(defaultPasswordBytes)
		if e != nil {
			return nil, fmt.Errorf("failed random password generation: %w", e)
		}
		password = hex.EncodeToString(pwd)
	}
	// db item prepare
	now := time.Now().UTC()
	item := &db.Item{
		Key:       p.Log.ID,
		Text:      text,
		FileName:  fileName,
		CountText: times,
		CountFile: times,
		Created:   now,
		Updated:   now,
		Expired:   now.Add(time.Duration(ttl) * time.Second),
		Storage:   p.Storage,
	}
	err = item.Encrypt(password, f)
	if err != nil {
		return nil, fmt.Errorf("failed encryption: %w", err)
	}
	return item, nil
}

// upload gets incoming data and saves it to the storage.
func upload(ctx context.Context, w http.ResponseWriter, p *Params) error {
	item, err := validateUpload(w, p)
	if err != nil {
		return err
	}
	if item == nil {
		// failed validation, it's already handled
		return nil
	}
	err = item.Save(ctx, p.DB)
	if err != nil {
		return err
	}

	return nil
}
