package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/homedepot/flop"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCurrentBin(t *testing.T) {
	home, err := copyTestData("validate")
	require.NoError(t, err)

	cfg := Config{Home: home, Name: "dummyd"}
	assert.Equal(t, cfg.GenesisBin(), cfg.CurrentBin())

	// ensure we cannot set this to an invalid value
	for _, name := range []string{"missing", "nobin", "noexec"} {
		err = cfg.SetCurrentUpgrade(name)
		require.Error(t, err, name)
		assert.Equal(t, cfg.GenesisBin(), cfg.CurrentBin(), name)
	}

	linkPath := filepath.Join(cfg.Root(), "current")
	binPath := filepath.Join(linkPath, "bin", "dummyd")
	activePath := cfg.UpgradeDir("chain2")

	// now set it to a valid upgrade and make sure CurrentBin is now set properly
	err = cfg.SetCurrentUpgrade("chain2")
	require.NoError(t, err)
	// we should see current in the path
	assert.Equal(t, binPath, cfg.CurrentBin())
	// make sure current points to current upgrade
	dest, err := os.Readlink(linkPath)
	require.NoError(t, err)
	require.Equal(t, activePath, dest)

	// // setting it a second time shouldn't fail either (test update symlink)
	// err = cfg.SetCurrentUpgrade("chain2")
	// require.NoError(t, err)
	// assert.Equal(t, cfg.UpgradeBin("chain2"), cfg.CurrentBin())
}

func TestDoUpgrade(t *testing.T) {
	home, err := copyTestData("validate")
	require.NoError(t, err)

	cfg := Config{Home: home, Name: "dummyd"}
	assert.Equal(t, cfg.GenesisBin(), cfg.CurrentBin())
}

// copyTestData will make a tempdir and then
// "cp -r" a subdirectory under testdata there
// returns the directory (which can now be used as Config.Home) and modified safely
func copyTestData(subdir string) (string, error) {
	tmpdir, err := ioutil.TempDir("", "upgrade-manager-test")
	if err != nil {
		return "", errors.Wrap(err, "create temp dir")
	}

	src := filepath.Join("testdata", subdir)
	options := flop.Options{
		Recursive: true,
		// MkdirAll:  true,
	}
	err = flop.Copy(src, tmpdir, options)
	if err != nil {
		os.RemoveAll(tmpdir)
		return "", errors.Wrap(err, "copying files")
	}
	return tmpdir, nil
}
