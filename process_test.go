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
	err = LaunchProcess(cfg, args, &stdout, &stderr)
	// err = LaunchProcess(cfg, args, os.Stdout, os.Stderr)
	require.NoError(t, err)
	assert.Equal(t, "", stderr.String())
	assert.Equal(t, "Genesis foo bar 1234\nUPGRADE \"chain2\" NEEDED at height 49: {}\n", stdout.String())

	// ensure this is upgraded now and produces new output
	require.Equal(t, cfg.UpgradeBin("chain2"), cfg.CurrentBin())
	args = []string{"second", "run", "--verbose"}
	stdout.Reset()
	stderr.Reset()
	err = LaunchProcess(cfg, args, &stdout, &stderr)
	require.NoError(t, err)
	assert.Equal(t, "", stderr.String())
	assert.Equal(t, "Chain 2 is live!\nArgs: second run --verbose\nFinished successfully\n", stdout.String())

	// ended without other upgrade
	require.Equal(t, cfg.UpgradeBin("chain2"), cfg.CurrentBin())
}

// TestLaunchProcess will try running the script a few times and watch upgrades work properly
// and args are passed through
func TestLaunchProcessWithDownloads(t *testing.T) {
	// this is a fun path
	// genesis -> "chain2" = zip_binary
	// zip_binary -> "chain3" = ref_zipped -> zip_directory
	// zip_directory no upgrade
	home, err := copyTestData("download")
	cfg := &Config{Home: home, Name: "autod", AllowDownloadBinaries: true}
	require.NoError(t, err)
	defer os.RemoveAll(home)

	// should run the genesis binary and produce expected output
	require.Equal(t, cfg.GenesisBin(), cfg.CurrentBin())
	var stdout, stderr bytes.Buffer
	args := []string{"some", "args"}
	err = LaunchProcess(cfg, args, &stdout, &stderr)
	require.NoError(t, err)
	assert.Equal(t, "", stderr.String())
	assert.Equal(t, "Preparing auto-download some args\n"+`UPGRADE "chain2" NEEDED at height 49: {"binaries": {"linux/amd64": "https://github.com/regen-network/cosmos-upgrade-manager/raw/auto-download/testdata/repo/zip_binary/autod.zip?checksum=sha256:1517886fb44425c30571316fcbd6380b38f80fb3dbaadc5ecd3303046b43b0cc"}}`+"\n", stdout.String())

	// ensure this is upgraded now and produces new output
	require.Equal(t, cfg.UpgradeBin("chain2"), cfg.CurrentBin())
	args = []string{"run", "--fast"}
	stdout.Reset()
	stderr.Reset()
	err = LaunchProcess(cfg, args, &stdout, &stderr)
	require.NoError(t, err)
	assert.Equal(t, "", stderr.String())
	assert.Equal(t, "Chain 2 from zipped binary link to referal\nArgs: run --fast\n"+`UPGRADE "chain3" NEEDED at height 936: https://github.com/regen-network/cosmos-upgrade-manager/raw/auto-download/testdata/repo/ref_zipped?checksum=sha256:59708d1142dfd3f9d6ffafecd7f89c641d982c9d5d999d86b211a1a736f114ad`+"\n", stdout.String())

	// ended with one more upgrade
	require.Equal(t, cfg.UpgradeBin("chain3"), cfg.CurrentBin())
	// make sure this is the proper binary now....
	args = []string{"end", "--halt"}
	stdout.Reset()
	stderr.Reset()
	err = LaunchProcess(cfg, args, &stdout, &stderr)
	require.NoError(t, err)
	assert.Equal(t, "", stderr.String())
	assert.Equal(t, "Chain 2 from zipped directory\nArgs: end --halt\n", stdout.String())

	// and this doesn't upgrade
	require.Equal(t, cfg.UpgradeBin("chain3"), cfg.CurrentBin())
}
