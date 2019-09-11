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
	defer os.RemoveAll(home)

	cfg := Config{Home: home, Name: "dummyd"}
	assert.Equal(t, cfg.GenesisBin(), cfg.CurrentBin())

	// ensure we cannot set this to an invalid value
	for _, name := range []string{"missing", "nobin", "noexec"} {
		err = cfg.SetCurrentUpgrade(name)
		require.Error(t, err, name)
		assert.Equal(t, cfg.GenesisBin(), cfg.CurrentBin(), name)
	}

	// try a few times to make sure this can be reproduced
	for _, upgrade := range []string{"chain2", "chain3", "chain2"} {
		// now set it to a valid upgrade and make sure CurrentBin is now set properly
		err = cfg.SetCurrentUpgrade(upgrade)
		require.NoError(t, err)
		// we should see current point to the new upgrade dir
		assert.Equal(t, cfg.UpgradeBin(upgrade), cfg.CurrentBin())
	}
}

func TestDoUpgrade(t *testing.T) {
	home, err := copyTestData("validate")
	require.NoError(t, err)
	defer os.RemoveAll(home)

	cfg := &Config{Home: home, Name: "dummyd"}
	assert.Equal(t, cfg.GenesisBin(), cfg.CurrentBin())

	// do upgrade ignores bad files
	for _, name := range []string{"missing", "nobin", "noexec"} {
		info := &UpgradeInfo{Name: name}
		err = DoUpgrade(cfg, info)
		require.Error(t, err, name)
		assert.Equal(t, cfg.GenesisBin(), cfg.CurrentBin(), name)
	}

	// make sure it updates a few times
	for _, upgrade := range []string{"chain2", "chain3"} {
		// now set it to a valid upgrade and make sure CurrentBin is now set properly
		info := &UpgradeInfo{Name: upgrade}
		err = DoUpgrade(cfg, info)
		require.NoError(t, err)
		// we should see current point to the new upgrade dir
		assert.Equal(t, cfg.UpgradeBin(upgrade), cfg.CurrentBin())
	}

	// TODO: test with download
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
