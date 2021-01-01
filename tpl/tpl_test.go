package tpl

import (
	"os"
	"path/filepath"
	"testing"
)

var htmlFolder = filepath.Join(os.TempDir(), "test_send", "html") // "make prepare" does it

func TestLoad(t *testing.T) {
	const numTemplates = 4
	tmpDir := filepath.Join(os.TempDir(), "send_tpl_test")
	err := os.Mkdir(tmpDir, 0700)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if e := os.RemoveAll(tmpDir); e != nil {
			t.Error(e)
		}
	}()
	_, err = Load(tmpDir)
	if err == nil {
		t.Error("unexpected successful result")
	}

	templates, err := Load(htmlFolder)
	if err != nil {
		t.Error(err)
	}
	if n := len(templates); n != numTemplates {
		t.Errorf("failed templates number %d", n)
	}
	for name, value := range templates {
		if value == nil {
			t.Errorf("nil template=%s value", name)
		}
	}
}
