package appconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds heoscli config: default device, format, and optional HEOS credentials.
type Config struct {
	DefaultDevice string `json:"defaultDevice,omitempty"`
	Format        string `json:"format,omitempty"`
	HEOSUsername  string `json:"heosUsername,omitempty"`
	HEOSPassword  string `json:"heosPassword,omitempty"`
}

// Normalize trims and sets defaults.
func (c Config) Normalize() Config {
	out := Config{
		DefaultDevice: strings.TrimSpace(c.DefaultDevice),
		Format:        strings.ToLower(strings.TrimSpace(c.Format)),
		HEOSUsername:  strings.TrimSpace(c.HEOSUsername),
		HEOSPassword:  c.HEOSPassword,
	}
	if out.Format == "" {
		out.Format = "plain"
	}
	if !isValidFormat(out.Format) {
		out.Format = "plain"
	}
	return out
}

func isValidFormat(format string) bool {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "plain", "json", "tsv":
		return true
	default:
		return false
	}
}

// Store loads and saves config.
type Store interface {
	Path() string
	Load() (Config, error)
	Save(cfg Config) error
}

// FileStore stores config in a JSON file.
type FileStore struct {
	path string
}

// NewFileStore returns a FileStore for the given path.
func NewFileStore(path string) (*FileStore, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("path is required")
	}
	return &FileStore{path: path}, nil
}

// NewDefaultStore returns a FileStore under UserConfigDir/heoscli/config.json.
func NewDefaultStore() (*FileStore, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return &FileStore{path: filepath.Join(dir, "heoscli", "config.json")}, nil
}

// Path returns the config file location.
func (s *FileStore) Path() string { return s.path }

type fileFormat struct {
	Version int    `json:"version"`
	Config  Config `json:"config"`
}

// Load reads the config file. Returns normalized default config if file does not exist.
func (s *FileStore) Load() (Config, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}.Normalize(), nil
		}
		return Config{}, err
	}
	var ff fileFormat
	if err := json.Unmarshal(raw, &ff); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	return ff.Config.Normalize(), nil
}

// Save writes the config file.
func (s *FileStore) Save(cfg Config) error {
	cfg = cfg.Normalize()
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	ff := fileFormat{Version: 1, Config: cfg}
	b, err := json.MarshalIndent(ff, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
