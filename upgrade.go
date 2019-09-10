package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"

	getter "github.com/hashicorp/go-getter"
	"github.com/pkg/errors"
)

// DoUpgrade will be called after the log message has been parsed and the process has terminated.
// We can now make any changes to the underlying directory without interferance and leave it
// in a state the will make a proper restart
func DoUpgrade(cfg *Config, info *UpgradeInfo) error {
	err := EnsureBinary(cfg.UpgradeBin(info.Name))

	// Simplest case is to switch the link
	if err == nil {
		// we have the binary - do it
		return cfg.SetCurrentUpgrade(info.Name)
	}

	// if auto-download is disabled, we fail
	if !cfg.Download {
		return errors.Wrap(err, "binary not present, downloading disabled")
	}
	// if the dir is there already, don't download either
	_, err = os.Stat(cfg.UpgradeDir(info.Name))
	if !os.IsNotExist(err) {
		return errors.Errorf("upgrade dir already exists, won't overwrite")
	}

	// If not there, then we try to download it... maybe
	if err := DownloadBinary(cfg, info); err != nil {
		return errors.Wrap(err, "cannot download binary")
	}

	// and then set the binary again
	err = EnsureBinary(cfg.UpgradeBin(info.Name))
	if err != nil {
		return errors.Wrap(err, "downloaded binary doesn't check out")
	}
	return cfg.SetCurrentUpgrade(info.Name)
}

// DownloadBinary will grab the binary and place it in the proper directory
func DownloadBinary(cfg *Config, info *UpgradeInfo) error {
	url, err := GetDownloadURL(info)
	if err != nil {
		return err
	}

	// download
	path := cfg.UpgradeDir(info.Name)
	return getter.GetFile(path, url)
}

// UpgradeConfig is expected format for the info field to allow auto-download
type UpgradeConfig struct {
	Binaries map[string]string `json:"binaries"`
}

// GetDownloadURL will check if there is an arch-dependent binary specified in Info
func GetDownloadURL(info *UpgradeInfo) (string, error) {
	// check if it is the upgrade config
	var config UpgradeConfig
	err := json.Unmarshal([]byte(info.Info), &config)
	if err == nil {
		url, ok := config.Binaries[osArch()]
		if !ok {
			return "", errors.Errorf("cannot find binary for os/arch: %s", osArch())
		}
		return url, nil
	}

	// TODO: download file then parse that
	return "", errors.New("upgrade info doesn't contain binary map")
}

func osArch() string {
	return fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
}
