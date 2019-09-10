package main

import (
	"net/url"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

const (
	rootName    = "upgrade_manager"
	genesisDir  = "genesis"
	upgradesDir = "upgrades"
	currentLink = "current"
)

// Config is the information passed in to control the daemon
type Config struct {
	Home     string
	Name     string
	Download bool
}

// Root returns the root directory where all info lives
func (cfg *Config) Root() string {
	return filepath.Join(cfg.Home, rootName)
}

// GenesisBin is the path to the genesis binary - must be in place to start manager
func (cfg *Config) GenesisBin() string {
	return filepath.Join(cfg.Root(), genesisDir, "bin", cfg.Name)
}

// UpgradeBin is the path to the binary for the named upgrade
func (cfg *Config) UpgradeBin(upgradeName string) string {
	return filepath.Join(cfg.UpgradeDir(upgradeName), "bin", cfg.Name)
}

// UpgradeDir is the directory named upgrade
func (cfg *Config) UpgradeDir(upgradeName string) string {
	safeName := url.PathEscape(upgradeName)
	return filepath.Join(cfg.Root(), upgradesDir, safeName)
}

// CurrentBin is the path to the currently selected binary (genesis if no link is set)
func (cfg *Config) CurrentBin() string {
	cur := filepath.Join(cfg.Root(), currentLink)
	// if nothing here, fallback to genesis
	if _, err := os.Stat(cur); err != nil {
		return cfg.GenesisBin()
	}
	return filepath.Join(cur, "bin", cfg.Name)
}

// SetCurrentUpgrade sets the named upgrade to be the current link, returns error if this binary doesn't exist
func (cfg *Config) SetCurrentUpgrade(upgradeName string) error {
	// ensure named upgrade exists
	bin := cfg.UpgradeBin(upgradeName)
	if err := EnsureBinary(bin); err != nil {
		return err
	}

	// set a symbolic link
	link := filepath.Join(cfg.Root(), currentLink)
	safeName := url.PathEscape(upgradeName)
	upgrade := filepath.Join(cfg.Root(), upgradesDir, safeName)
	if err := os.Symlink(link, upgrade); err != nil {
		return errors.Wrap(err, "creating current symlink")
	}
	return nil
}

// EnsureBinary ensures the file exists and is executable, or returns an error
func EnsureBinary(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return errors.Wrap(err, "cannot stat home dir")
	}
	if !info.Mode().IsRegular() {
		return errors.Errorf("%s is not a regular file", info.Name())
	}
	// this checks if the world-executable bit is set (we cannot check owner easily)
	exec := info.Mode().Perm() & 0001
	if exec == 0 {
		return errors.Errorf("%s is not world executable", info.Name())
	}
	return nil
}

// GetConfigFromEnv will read the environmental variables into a config
// and then validate it is reasonable
func GetConfigFromEnv() (*Config, error) {
	cfg := &Config{
		Home: os.Getenv("DAEMON_HOME"),
		Name: os.Getenv("DAEMON_NAME"),
	}
	if os.Getenv("DAEMON_DOWNLOAD") == "on" {
		cfg.Download = true
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// validate returns an error if this config is invalid.
// it enforces Home/upgrade_manager is a valid directory and exists,
// and that Name is set
func (cfg *Config) validate() error {
	if cfg.Name == "" {
		return errors.New("DAEMON_NAME is not set")
	}
	if cfg.Home == "" {
		return errors.New("DAEMON_HOME is not set")
	}

	// ensure the root directory exists
	info, err := os.Stat(cfg.Root())
	if err != nil {
		return errors.Wrap(err, "cannot stat home dir")
	}
	if !info.IsDir() {
		return errors.Errorf("%s is not a directory", info.Name())
	}

	return nil
}
