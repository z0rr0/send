package cfg

import (
	"html/template"
	"path/filepath"
)

// html templates names
const (
	Index  = "index"
	Upload = "upload"
)

func parseTemplates(fullPath string) (map[string]*template.Template, error) {
	base := filepath.Join(fullPath, "base.html")
	index, err := template.ParseFiles(base, filepath.Join(fullPath, "index.html"))
	if err != nil {
		return nil, err
	}

	upload, err := template.ParseFiles(base, filepath.Join(fullPath, "upload.html"))
	if err != nil {
		return nil, err
	}
	return map[string]*template.Template{Index: index, Upload: upload}, nil
}
