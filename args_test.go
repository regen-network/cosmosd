package main

import (
	"testing"

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