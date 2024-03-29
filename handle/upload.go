package handle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/z0rr0/send/cfg"
	"github.com/z0rr0/send/db"
	"github.com/z0rr0/send/encrypt/pwgen"
)

// UploadData is upload result page data.
type UploadData struct {
	URL        string `json:"url"`
	Password   string `json:"password"`
	PwdDisable bool   `json:"pwd_disable"`
	code       int
}

// isValid returns true if validation is ok.
func (u *UploadData) isValid() bool {
	return u.code >= http.StatusOK && u.code < http.StatusMultipleChoices
}

type validUploadData struct {
	item     *db.Item
	password string
	code     int
}

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
func failedUpload(w http.ResponseWriter, code int, data *IndexData, p *Params, isAPI bool) error {
	var err error
	w.WriteHeader(code)
	if isAPI {
		encoder := json.NewEncoder(w)
		return encoder.Encode(&ErrItem{Err: data.Error})
	}
	err = p.Settings.Tpl[cfg.IndexTpl].ExecuteTemplate(w, cfg.IndexTpl, data)
	if err != nil {
		return fmt.Errorf("failed execute template=%s: %w", cfg.IndexTpl, err)
	}
	return nil
}

// validateUpload checks incoming request data
// and returns new db.Item pointer, password and validation error.
func validateUpload(w http.ResponseWriter, p *Params, isAPI bool) (*validUploadData, error) {
	var (
		fileMeta             string
		autoPassword         bool
		countText, countFile int
	)
	data := &IndexData{MaxSize: p.Settings.Size}
	vd := &validUploadData{code: http.StatusBadRequest}
	if p.Request.Method != "POST" {
		data.Error = "failed HTTP method"
		vd.code = http.StatusMethodNotAllowed
		return vd, failedUpload(w, vd.code, data, p, isAPI)
	}
	// file
	f, h, err := p.Request.FormFile("file")
	if err != nil {
		if !errors.Is(err, http.ErrMissingFile) {
			data.Error = "failed file upload"
			return vd, failedUpload(w, vd.code, data, p, isAPI)
		}
		// ErrMissingFile will be checked later with text-field
	} else {
		err = p.Storage.Limit(h.Size)
		if err != nil {
			data.Error = "no space in file storage"
			p.Log.Error("%s: %v", data.Error, err)
			return vd, failedUpload(w, vd.code, data, p, isAPI)
		}
		fm := &FileMeta{Name: h.Filename, Size: h.Size, ContentType: h.Header.Get("Content-Type")}
		fileMeta, err = fm.Encode()
		if err != nil {
			return nil, err
		}
	}
	defer func() {
		if e := p.Request.Body.Close(); e != nil {
			p.Log.Error("close request body: %v", e)
		}
		if fileMeta != "" {
			if e := f.Close(); e != nil {
				p.Log.Error("close incoming file: %v", e)
			}
		}
	}()
	// text
	text := p.Request.PostFormValue("text")
	if fileMeta == "" && text == "" {
		data.Error = "empty text and file fields"
		return vd, failedUpload(w, vd.code, data, p, isAPI)
	}
	// ttl
	ttl, err := validateInt("TTL", p.Request.PostFormValue("ttl"), p.Settings.TTL)
	if err != nil {
		data.Error = "incorrect TTL"
		return vd, failedUpload(w, vd.code, data, p, isAPI)
	}
	// times
	times, err := validateInt("times", p.Request.PostFormValue("times"), p.Settings.Times)
	if err != nil {
		data.Error = "incorrect times"
		return vd, failedUpload(w, vd.code, data, p, isAPI)
	}
	// password
	password := p.Request.PostFormValue("password")
	if password == "" {
		// auto generation
		password = pwgen.New(p.Settings.PassLen)
		autoPassword = true
	}
	// db item prepare
	switch {
	case fileMeta == "":
		countText, countFile = times, 0
	case text == "":
		countText, countFile = 0, times
	default:
		countText, countFile = times, times
	}
	now := time.Now().UTC()
	item := &db.Item{
		Key:          p.Log.ID,
		Text:         text,
		FileMeta:     fileMeta,
		CountText:    countText,
		CountMeta:    countText + countFile, // can be read twice for text and file
		CountFile:    countFile,
		Created:      now,
		Updated:      now,
		Expired:      now.Add(time.Duration(ttl) * time.Second),
		Storage:      p.Storage.Dir,
		AutoPassword: autoPassword,
	}
	err = item.Encrypt(password, f)
	if err != nil {
		return nil, fmt.Errorf("failed encryption: %w", err)
	}
	vd.item = item
	vd.code = http.StatusCreated
	vd.password = password
	return vd, nil
}

// uploadCommon is a handler for API and web upload methods.
func uploadCommon(ctx context.Context, w http.ResponseWriter, p *Params, isAPI bool) (*UploadData, error) {
	validData, err := validateUpload(w, p, isAPI)
	if err != nil {
		return nil, err
	}
	data := &UploadData{code: validData.code, Password: validData.password}
	if validData.item == nil {
		// failed validation, it's already handled
		return data, nil
	}
	err = validData.item.Save(ctx, p.DB)
	if err != nil {
		return nil, err
	}
	if !validData.item.AutoPassword {
		data.Password = "*****"
		data.PwdDisable = true
	}
	data.URL = validData.item.GetURL(p.Request, p.Secure).String()
	return data, nil
}

// uploadHandler gets incoming data and saves it to the storage.
func uploadHandler(ctx context.Context, w http.ResponseWriter, p *Params) (int, error) {
	data, err := uploadCommon(ctx, w, p, false)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	if !data.isValid() {
		return data.code, nil
	}
	err = p.Settings.Tpl[cfg.UploadTpl].ExecuteTemplate(w, cfg.UploadTpl, data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed execute template=%s: %w", cfg.UploadTpl, err)
	}
	return data.code, nil
}
