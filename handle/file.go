package handle

import "encoding/json"

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

// DecodeMeta returns a parsed from json string file metadata.
func DecodeMeta(fileMeta string) (*FileMeta, error) {
	f := &FileMeta{}
	err := json.Unmarshal([]byte(fileMeta), f)
	if err != nil {
		return nil, err
	}
	return f, nil
}
