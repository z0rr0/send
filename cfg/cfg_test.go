package cfg

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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
	_, err := New("/bad_file_path.json", nil)
	if err == nil {
		t.Fatal("unexpected behavior")
	}
	c, err := New(testConfig, nil)
	if err != nil {
		t.Fatalf("failed read config: %v", err)
	}
	err = c.Close()
	if err != nil {
		t.Errorf("close error: %v", err)
	}
}

func TestConfig(t *testing.T) {
	c, err := New(testConfig, nil)
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
	if timeout := c.Timeout(); timeout != 30*time.Second {
		t.Errorf("failed timeout: %v", timeout)
	}
	if timeout := c.GCPeriod(); timeout != 10*time.Second {
		t.Errorf("failed gc period: %v", timeout)
	}
	if timeout := c.Shutdown(); timeout != 5*time.Second {
		t.Errorf("failed shutdown: %v", timeout)
	}
	if secret := c.Secret("xyz"); secret != "xyzabc" {
		t.Errorf("failed secret: %v", secret)
	}
}
