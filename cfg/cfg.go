package cfg

// Package cfg contains structures and functions for configurations reading and validation.

import (
	"database/sql"
	"fmt"
	"html/template"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite3 driver package
	"github.com/pelletier/go-toml"
)

// html templates names
const (
	BaseTpl     = "base.html"
	IndexTpl    = "index.html"
	UploadTpl   = "upload.html"
	DownloadTpl = "download.html"
	ErrorTpl    = "error.html"
)

type server struct {
	Host    string `toml:"host"`
	Port    int    `toml:"port"`
	Timeout int    `toml:"timeout"`
	Secure  bool   `toml:"secure"`
}

type storage struct {
	File    string `toml:"file"`
	Dir     string `toml:"dir"`
	Timeout int    `toml:"timeout"`
	Db      *sql.DB
}

// String returns base info about storage.
func (s *storage) String() string {
	return fmt.Sprintf("database=%s, files=%s", s.File, s.Dir)
}

// Settings is base service settings.
type Settings struct {
	TTL       int                           `toml:"ttl"`
	Times     int                           `toml:"times"`
	Size      int                           `toml:"size"`
	Salt      string                        `toml:"salt"`
	GC        int                           `toml:"gc"`
	PassLen   int                           `toml:"passlen"`
	Shutdown  int                           `toml:"shutdown"`
	Templates string                        `toml:"templates"`
	Static    string                        `toml:"static"`
	Tpl       map[string]*template.Template `toml:"-"`
}

// Config is a main configuration structure.
type Config struct {
	Server   server   `toml:"server"`
	Storage  storage  `toml:"storage"`
	Settings Settings `toml:"settings"`
}

// Addr returns service's net address.
func (c *Config) Addr() string {
	return net.JoinHostPort(c.Server.Host, fmt.Sprint(c.Server.Port))
}

// Close frees resources.
func (c *Config) Close() error {
	return c.Storage.Db.Close()
}

// Timeout is service timeout.
func (c *Config) Timeout() time.Duration {
	return time.Duration(c.Server.Timeout) * time.Second
}

// GCPeriod is gc period in seconds.
func (c *Config) GCPeriod() time.Duration {
	return time.Duration(c.Settings.GC) * time.Second
}

// DbPeriod is gc database period in seconds.
func (c *Config) DbPeriod() time.Duration {
	return time.Duration(c.Storage.Timeout) * time.Second
}

// MaxFileSize returns max file size.
func (c *Config) MaxFileSize() int {
	return c.Settings.Size << 20
}

// Secret returns string with salt.
func (c *Config) Secret(p string) string {
	return p + c.Settings.Salt
}

// Shutdown is shutdown timeout.
func (c *Config) Shutdown() time.Duration {
	return time.Duration(c.Settings.Shutdown) * time.Second
}

// isValid checks the Settings are valid.
func (c *Config) isValid() error {
	const (
		userReadWrite  os.FileMode = 0600
		userReadSearch os.FileMode = 0500
	)
	fullPath, err := checkDirectory(c.Settings.Templates, userReadSearch)
	if err != nil {
		return err
	}
	tpl, err := parseTemplates(fullPath)
	if err != nil {
		return err
	}
	c.Settings.Templates = fullPath
	c.Settings.Tpl = tpl

	fullPath, err = checkDirectory(c.Storage.Dir, userReadWrite)
	if err != nil {
		return err
	}
	c.Storage.Dir = fullPath

	fullPath, err = checkDirectory(c.Settings.Static, userReadSearch)
	if err != nil {
		return err
	}
	c.Settings.Static = fullPath

	err = isGreaterThanZero(c.Storage.Timeout, "storage.timeout", err)
	err = isGreaterThanZero(c.Server.Timeout, "server.timeout", err)
	err = isGreaterThanZero(c.Server.Port, "server.port", err)
	err = isGreaterThanZero(c.Settings.TTL, "settings.ttl", err)
	err = isGreaterThanZero(c.Settings.Times, "settings.times", err)
	err = isGreaterThanZero(c.Settings.Size, "settings.size", err)
	err = isGreaterThanZero(c.Settings.GC, "settings.gc", err)
	err = isGreaterThanZero(c.Settings.PassLen, "settings.gc", err)
	err = isGreaterThanZero(c.Settings.Shutdown, "settings.shutdown", err)
	return err
}

// New returns new configuration.
func New(filename string) (*Config, error) {
	fullPath, err := filepath.Abs(strings.Trim(filename, " "))
	if err != nil {
		return nil, fmt.Errorf("config file: %w", err)
	}
	_, err = os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("config existing: %w", err)
	}
	data, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("config read: %w", err)
	}
	c := &Config{}
	err = toml.Unmarshal(data, c)
	if err != nil {
		return nil, fmt.Errorf("config parsing: %w", err)
	}
	err = c.isValid()
	if err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}
	database, err := sql.Open("sqlite3", c.Storage.File)
	if err != nil {
		return nil, fmt.Errorf("db file: %w", err)
	}
	c.Storage.Db = database
	return c, nil
}

// isGreaterThanZero returns error if err is already error or x is less than 1.
func isGreaterThanZero(x int, name string, err error) error {
	if err != nil {
		return err
	}
	if x < 1 {
		return fmt.Errorf("%s=%d should be greater than 1", name, x)
	}
	return nil
}

// checkDirectory checks that dir exists and it is a directory with correct permissions.
func checkDirectory(name string, mode os.FileMode) (string, error) {
	fullPath, err := filepath.Abs(strings.Trim(name, " "))
	if err != nil {
		return "", err
	}
	info, err := os.Stat(fullPath)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("fs object '%s' is not a directory", name)
	}
	if m := info.Mode().Perm(); m&mode != mode {
		return "", fmt.Errorf("directory '%s' has failed permissions=%o, mode=%o", name, m, mode)
	}
	return fullPath, nil
}

// parseTemplates parses expected templates files.
func parseTemplates(fullPath string) (map[string]*template.Template, error) {
	base := filepath.Join(fullPath, BaseTpl)
	templates := []string{IndexTpl, UploadTpl, DownloadTpl, ErrorTpl}
	templateMap := make(map[string]*template.Template)
	for _, name := range templates {
		tpl, err := template.ParseFiles(base, filepath.Join(fullPath, name))
		if err != nil {
			return nil, fmt.Errorf("failed parse template %s: %w", name, err)
		}
		templateMap[name] = tpl
	}
	return templateMap, nil
}
