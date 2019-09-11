package main

import (
	"testing"
	"path/filepath"

	"github.com/stretchr/testify/assert"
)

// Note all tests use the / path separator for simplicity and will fail on windows.
// The actual code should work on windows... probably (not sure about process stuff)

func TestConfigPaths(t *testing.T) {
	cases := map[string]struct{
		cfg Config 
		upgradeName string
		expectRoot string 
		expectGenesis string
		expectUpgrade string
	}{
		"simple": {
			cfg: Config{Home: "/foo", Name: "myd"},
			upgradeName: "bar",
			expectRoot: "/foo/upgrade_manager",
			expectGenesis: "/foo/upgrade_manager/genesis/bin/myd",
			expectUpgrade: "/foo/upgrade_manager/upgrades/bar/bin/myd",
		},
		"handle space": {
			cfg: Config{Home: "/longer/prefix/", Name: "yourd"},
			upgradeName: "some spaces",
			expectRoot: "/longer/prefix/upgrade_manager",
			expectGenesis: "/longer/prefix/upgrade_manager/genesis/bin/yourd",
			expectUpgrade: "/longer/prefix/upgrade_manager/upgrades/some%20spaces/bin/yourd",
		},
	}

	for name, tc := range cases {
		t.Run(name, func (t *testing.T) {
			assert.Equal(t, tc.cfg.Root(), tc.expectRoot)
			assert.Equal(t, tc.cfg.GenesisBin(), tc.expectGenesis)
			assert.Equal(t, tc.cfg.UpgradeBin(tc.upgradeName), tc.expectUpgrade)
		})
	}
}

// Test validate
func TestValidate(t *testing.T) {
	relPath := filepath.Join("testdata", "validate")
	absPath, err := filepath.Abs(relPath)
	assert.NoError(t, err)

	testdata, err := filepath.Abs("testdata")
	assert.NoError(t, err)

	cases := map[string]struct{
		cfg Config 
		valid bool
	}{
		"happy": {
			cfg: Config{Home: absPath, Name: "bind"},
			valid: true,
		},
		"happy with download": {
			cfg: Config{Home: absPath, Name: "bind", Download: true},
			valid: true,
		},
		"missing home": {
			cfg: Config{Name: "bind"},
			valid: false,
		},
		"missing name": {
			cfg: Config{Home: absPath},
			valid: false,
		},
		"relative path": {
			cfg: Config{Home: relPath, Name: "bind"},
			valid: false,
		},
		"no upgrade manager subdir": {
			cfg: Config{Home: testdata, Name: "bind"},
			valid: false,
		},
		"no such dir": {
			cfg: Config{Home: "/no/such/dir", Name: "bind"},
			valid: false,
		},

	}

	for name, tc := range cases {
		t.Run(name, func (t *testing.T) {
			err := tc.cfg.validate()
			if tc.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestEnsureBin(t *testing.T) {
	relPath := filepath.Join("testdata", "validate")
	absPath, err := filepath.Abs(relPath)
	assert.NoError(t, err)

	cfg := Config{Home: absPath, Name: "dummyd"}
	assert.NoError(t, cfg.validate())

	err = EnsureBinary(cfg.GenesisBin())
	assert.NoError(t, err)

	cases := map[string]struct{
		upgrade string
		hasBin bool
	}{
		"proper": {"chain2", true},
		"no binary": {"nobin", false},
		"not executable": {"noexec", false},
		"no directory": {"foobarbaz", false},
	}

	for name, tc := range cases {
		t.Run(name, func (t *testing.T) {
			err := EnsureBinary(cfg.UpgradeBin(tc.upgrade))
			if tc.hasBin {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// setup test data and try out EnsureBinary with various file setups and permissions, also CurrentBin/SetCurrentBin