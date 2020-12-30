package cfg

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	// test config file name
	cfgName = "test_send.toml"
)

var (
	// test config full path, defined in Makefile (env TMPDIR)
	testConfig = filepath.Join(os.TempDir(), cfgName)
)

func TestNew(t *testing.T) {
	_, err := New("/bad_file_path.json")
	if err == nil {
		t.Fatal("unexpected behavior")
	}
	c, err := New(testConfig)
	if err != nil {
		t.Fatalf("failed read config: %v", err)
	}
	if c.Addr() == "" {
		t.Error("empty address")
	}
	c.Settings.Size = 4
	if m := c.MaxFileSize(); m != (1048576 * 4) {
		t.Errorf("failed max size: %v", m)
	}
	err = c.Close()
	if err != nil {
		t.Errorf("close error: %v", err)
	}
}
