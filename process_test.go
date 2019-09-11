package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLaunchProcess will try running the script a few times and watch upgrades work properly
// and args are passed through
func TestLaunchProcess(t *testing.T) {
	home, err := copyTestData("validate")
	cfg := &Config{Home: home, Name: "dummyd"}
	require.NoError(t, err)
	defer os.RemoveAll(home)

	// should run the genesis binary and produce expected output
	require.Equal(t, cfg.GenesisBin(), cfg.CurrentBin())
	var stdout, stderr bytes.Buffer
	args := []string{"foo", "bar", "1234"}
	// err = LaunchProcess(cfg, args, &stdout, &stderr)
	err = LaunchProcess(cfg, args, os.Stdout, os.Stderr)
	require.NoError(t, err)
	assert.Equal(t, "", stderr.String())
	assert.Equal(t, "Genesis foo bar 1234\nUPGRADE \"chain2\" NEEDED at height 49: {}\n", stdout.String())

	// ensure this is upgraded now and produces new output
	require.Equal(t, cfg.UpgradeBin("chain2"), cfg.CurrentBin())
	args = []string{"second", "run", "--verbose"}
	err = LaunchProcess(cfg, args, &stdout, &stderr)
	require.NoError(t, err)
	assert.Equal(t, "", stderr.String())
	assert.Equal(t, "Chain 2 is live!\nArgs: second run --help\nFinished successfully\n", stdout.String())

	// ended without other upgrade
	require.Equal(t, cfg.UpgradeBin("chain2"), cfg.CurrentBin())
}
