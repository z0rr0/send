package tpl

// Package tpl loads and parses HTML templates from a target folder.

import (
	"context"
	"errors"
	"html/template"
	"path/filepath"
)

// keyType is custom type for context key.
type keyType uint8

// Templates is alias for templates map.
type Templates map[string]*template.Template

// key is context value key
const key keyType = 1

// Template names are files without extensions.
const (
	Index    = "index"
	Error    = "error"
	Upload   = "upload"
	Download = "download"
)

var (
	// ErrTplContext is an error when context value was not found.
	ErrTplContext = errors.New("not found tpl context")
	// ErrTplNotFound is an error when a tpl is not found
	ErrTplNotFound = errors.New("tpl not found")
)

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

// Set adds a map of templates to context.
func Set(ctx context.Context, templates Templates) context.Context {
	return context.WithValue(ctx, key, templates)
}

// Get returns a map of templates from context ctx.
func Get(ctx context.Context) (Templates, error) {
	v := ctx.Value(key)
	if v == nil {
		return nil, ErrTplContext
	}
	return v.(Templates), nil
}

// GetByName returns a tpl by its name from the context ctx.
func GetByName(ctx context.Context, name string) (*template.Template, error) {
	templates, err := Get(ctx)
	if err != nil {
		return nil, err
	}
	t, ok := templates[name]
	if !ok {
		return nil, ErrTplNotFound
	}
	return t, nil
}
