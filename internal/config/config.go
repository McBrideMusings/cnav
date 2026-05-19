package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// ProjectOverride is per-project user customization keyed by absolute CWD.
type ProjectOverride struct {
	CWD    string `toml:"cwd"`
	Hidden bool   `toml:"hidden,omitempty"`
	Name   string `toml:"name,omitempty"`
}

// Config is the on-disk state at ~/.config/cnav/config.toml.
type Config struct {
	Projects []ProjectOverride `toml:"project"`

	path  string
	index map[string]int // cwd -> index into Projects
}

// Load reads the config file. A missing file is not an error — Load returns an
// empty config bound to the default path so a subsequent Save creates it.
// Malformed files emit a stderr warning and produce an empty (but bound) config.
func Load() (*Config, error) {
	path, err := defaultPath()
	if err != nil {
		return nil, err
	}
	c := &Config{path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			c.rebuildIndex()
			return c, nil
		}
		return nil, err
	}
	if err := toml.Unmarshal(data, c); err != nil {
		fmt.Fprintf(os.Stderr, "cnav: ignoring malformed %s: %v\n", path, err)
		c.Projects = nil
	}
	c.rebuildIndex()
	return c, nil
}

// Save writes the config atomically. Records with no active overrides are
// pruned. The parent directory is created if missing.
func (c *Config) Save() error {
	c.prune()
	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return err
	}
	data, err := toml.Marshal(c)
	if err != nil {
		return err
	}
	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, c.path)
}

// Lookup returns the override for the given cwd. The zero value (no overrides)
// is returned when no record exists.
func (c *Config) Lookup(cwd string) ProjectOverride {
	if c == nil {
		return ProjectOverride{}
	}
	if i, ok := c.index[cwd]; ok {
		return c.Projects[i]
	}
	return ProjectOverride{}
}

// SetHidden upserts the Hidden flag for cwd. The caller is expected to Save.
func (c *Config) SetHidden(cwd string, hidden bool) {
	c.upsert(cwd).Hidden = hidden
}

// SetName upserts the custom Name for cwd. The caller is expected to Save.
func (c *Config) SetName(cwd, name string) {
	c.upsert(cwd).Name = name
}

// upsert returns a pointer to the record for cwd, creating one if absent.
func (c *Config) upsert(cwd string) *ProjectOverride {
	if i, ok := c.index[cwd]; ok {
		return &c.Projects[i]
	}
	c.Projects = append(c.Projects, ProjectOverride{CWD: cwd})
	c.index[cwd] = len(c.Projects) - 1
	return &c.Projects[len(c.Projects)-1]
}

// prune drops records that no longer carry any override.
func (c *Config) prune() {
	out := c.Projects[:0]
	for _, p := range c.Projects {
		if p.CWD == "" {
			continue
		}
		if !p.Hidden && p.Name == "" {
			continue
		}
		out = append(out, p)
	}
	c.Projects = out
	c.rebuildIndex()
}

func (c *Config) rebuildIndex() {
	c.index = make(map[string]int, len(c.Projects))
	for i, p := range c.Projects {
		c.index[p.CWD] = i
	}
}

func defaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "cnav", "config.toml"), nil
}
