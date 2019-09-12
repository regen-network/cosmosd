package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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

// TODO: test with download (and test all download functions)
func TestDoUpgradeNoDownloadUrl(t *testing.T) {
	home, err := copyTestData("validate")
	require.NoError(t, err)
	defer os.RemoveAll(home)

	cfg := &Config{Home: home, Name: "dummyd", AllowDownloadBinaries: true}
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
}

func TestOsArch(t *testing.T) {
	// all download tests will fail if we are not on linux...
	assert.Equal(t, "linux/amd64", osArch())
}

func TestGetDownloadURL(t *testing.T) {
	cases := map[string]struct {
		info  string
		url   string
		isErr bool
	}{
		"missing": {
			isErr: true,
		},
		"bad format": {
			info:  "https://foo.bar/",
			isErr: true,
		},
		"proper binary": {
			info: `{"binaries": {"linux/amd64": "https://foo.bar/", "windows/amd64": "https://something.else"}}`,
			url:  "https://foo.bar/",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			url, err := GetDownloadURL(&UpgradeInfo{Info: tc.info})
			if tc.isErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.url, url)
			}
		})
	}
}

func TestDownloadBinary(t *testing.T) {
	cases := map[string]struct {
		url         string
		canDownload bool
		validBinary bool
	}{
		"get raw binary": {
			url:         "./testdata/repo/raw_binary/autod",
			canDownload: true,
			validBinary: true,
		},
		"get raw binary with checksum": {
			// sha256sum ./testdata/repo/raw_binary/autod
			url:         "./testdata/repo/raw_binary/autod?checksum=sha256:e6bc7851600a2a9917f7bf88eb7bdee1ec162c671101485690b4deb089077b0d",
			canDownload: true,
			validBinary: true,
		},
		"get raw binary with invalid checksum": {
			url:         "./testdata/repo/raw_binary/autod?checksum=sha256:73e2bd6cbb99261733caf137015d5cc58e3f96248d8b01da68be8564989dd906",
			canDownload: false,
		},
		"get zipped directory": {
			url:         "./testdata/repo/zip_directory/autod.zip",
			canDownload: true,
			validBinary: true,
		},
		"get zipped directory with valid checksum": {
			// sha256sum ./testdata/repo/zip_directory/autod.zip
			url:         "./testdata/repo/zip_directory/autod.zip?checksum=sha256:29139e1381b8177aec909fab9a75d11381cab5adf7d3af0c05ff1c9c117743a7",
			canDownload: true,
			validBinary: true,
		},
		"get zipped directory with invalid checksum": {
			url:         "./testdata/repo/zip_directory/autod.zip?checksum=sha256:73e2bd6cbb99261733caf137015d5cc58e3f96248d8b01da68be8564989dd906",
			canDownload: false,
		},
		"invalid url": {
			url:         "./testdata/repo/bad_dir/autod",
			canDownload: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// make temp dir
			home, err := copyTestData("download")
			require.NoError(t, err)
			defer os.RemoveAll(home)

			cfg := &Config{
				Home:                  home,
				Name:                  "autod",
				AllowDownloadBinaries: true,
			}

			// if we have a relative path, make it absolute, but don't change eg. https://... urls
			url := tc.url
			if strings.HasPrefix(url, "./") {
				url, err = filepath.Abs(url)
				require.NoError(t, err)
			}

			upgrade := "amazonas"
			info := &UpgradeInfo{
				Name:   upgrade,
				Height: 789,
				Info:   fmt.Sprintf(`{"binaries":{"%s": "%s"}}`, osArch(), url),
			}

			err = DownloadBinary(cfg, info)
			if !tc.canDownload {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			err = EnsureBinary(cfg.UpgradeBin(upgrade))
			if tc.validBinary {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
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
		// this is set as workaround for https://github.com/homedepot/flop/issues/17
		Atomic: true,
	}
	err = flop.Copy(src, tmpdir, options)
	if err != nil {
		os.RemoveAll(tmpdir)
		return "", errors.Wrap(err, "copying files")
	}
	return tmpdir, nil
}
