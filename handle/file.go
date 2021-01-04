package handle

import (
	"encoding/json"
	"fmt"
	"strconv"
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
