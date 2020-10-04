package cfg

import (
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite3 driver package
	"github.com/pelletier/go-toml"
)

type server struct {
	Host    string `toml:"host"`
	Port    int    `toml:"port"`
	Timeout int    `toml:"timeout"`
	Secure  bool   `toml:"secure"`
}

type storage struct {
	File string `toml:"file"`
	Dir  string `toml:"dir"`
	Db   *sql.DB
}

type settings struct {
	TTL   int    `toml:"ttl"`
	Times int    `toml:"times"`
	Size  int    `toml:"size"`
	Salt  string `toml:"salt"`
	GC    int    `toml:"gc"`
}

// Config is configuration struct.
type Config struct {
	Server   server   `toml:"server"`
	Storage  storage  `toml:"storage"`
	Settings settings `toml:"settings"`
}

// Addr returns service's net address.
func (c *Config) Addr() string {
	return net.JoinHostPort(c.Server.Host, fmt.Sprint(c.Server.Port))
}

// Close frees resources.
func (c *Config) Close() error {
	return c.Storage.Db.Close()
}

// Timeout is service timeout in seconds.
func (c *Config) Timeout() time.Duration {
	return time.Duration(c.Settings.TTL) * time.Second
}

// GCPeriod is gc period in seconds.
func (c *Config) GCPeriod() time.Duration {
	return time.Duration(c.Settings.GC) * time.Second
}

// MaxFileSize returns max file size.
func (c *Config) MaxFileSize() int {
	return c.Settings.Size << 20
}

// Secret returns string with salt.
func (c *Config) Secret(p string) string {
	return p + c.Settings.Salt
}

// isValid checks the settings are valid.
func (c *Config) isValid() error {
	fullPath, err := filepath.Abs(strings.Trim(c.Storage.Dir, " "))
	if err != nil {
		return err
	}
	info, err := os.Stat(fullPath)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("storage is not a directory")
	}
	mode := uint(info.Mode().Perm())
	if mode&uint(0600) != 0600 {
		return errors.New("storage dir is not writable or readable")
	}
	c.Storage.Dir = fullPath

	err = isMoreThanZero(c.Server.Timeout, "server.timeout", err)
	err = isMoreThanZero(c.Server.Port, "server.port", err)
	err = isMoreThanZero(c.Settings.TTL, "settings.ttl", err)
	err = isMoreThanZero(c.Settings.Times, "settings.times", err)
	err = isMoreThanZero(c.Settings.Size, "settings.size", err)
	err = isMoreThanZero(c.Settings.GC, "settings.gc", err)
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
	database, err := sql.Open("sqlite3", c.Storage.File)
	if err != nil {
		return nil, fmt.Errorf("db file: %w", err)
	}
	c.Storage.Db = database
	return c, nil
}

// isMoreThanZero returns error if err is already error or x is less than 1.
func isMoreThanZero(x int, name string, err error) error {
	if err != nil {
		return err
	}
	if x < 1 {
		return fmt.Errorf("%s=%d should be more than 1", name, x)
	}
	return nil
}