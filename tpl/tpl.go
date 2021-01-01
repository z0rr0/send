package tpl

// Package tpl loads and parses HTML templates from a target folder.

import (
	"html/template"
	"path/filepath"
)

// Template names are files without extensions.
const (
	Index    = "index"
	Error    = "error"
	Upload   = "upload"
	Download = "download"
)

// Templates is alias for templates map.
type Templates map[string]*template.Template

// Load loads known templates to a map.
func Load(folder string) (Templates, error) {
	result := map[string]*template.Template{Index: nil, Error: nil, Upload: nil, Download: nil}
	for name := range result {
		fileName := filepath.Join(folder, name+".html")
		tpl, err := template.ParseFiles(fileName)
		if err != nil {
			return nil, err
		}
		result[name] = tpl
	}
	return result, nil
}
