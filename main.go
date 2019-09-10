package main

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/go-getter"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
)

type proc struct {
	args      []string
	cmd       *exec.Cmd
	upgrading bool
}

var upgradeRegex = regexp.MustCompile(`UPGRADE "(.*)" NEEDED at height (\d+): (.*)$`)

type upgradeListener struct {
	proc   *proc
	writer io.Writer
}

type UpgradeConfig struct {
	// Binaries is a map of OS/architecture names
	// to binary URI's that can be resolved with go-getter
	// (they should include SHA256 or SHA512 check-sums), or
	// it is a string that points to a JSON file with an os/architecture
	// to binary map. OS/architecture names are formed by concatenating
	// GOOS and GOARCH with "/" as the separator.
	Binaries map[string]string `json:"binaries"`
}

func (listener upgradeListener) Write(p []byte) (n int, err error) {
	matches := upgradeRegex.FindAll(p, 0)
	if matches != nil {
		if len(matches) != 3 {
			panic(fmt.Errorf("unexpected upgrade string: %s", p))
		}
		name := string(matches[0])
		// first check if there is a binary in upgrade_manager/
		binPath, err := bootstrapBinary(name, "", false)
		if err != nil {
			info := matches[2]
			configFile := filepath.Join(getDataDir(), "upgrades",
				fmt.Sprintf("%s.json", url.PathEscape(name)))
			if _, err := url.Parse(string(info)); err == nil {
				err = getter.GetFile(configFile, string(info))
				if err != nil {
					panic(err)
				}
			} else {
				err = ioutil.WriteFile(configFile, info, 0644)
				if err != nil {
					panic(err)
				}
			}
			binPath, err = processUpgradeConfig(name, configFile)
			if err != nil {
				panic(err)
			}
		}
		listener.proc.launchProc(binPath)
	}
	return listener.writer.Write(p)
}

func processUpgradeConfig(upgradeName, configFile string) (string, error) {
	src, err := ioutil.ReadFile(configFile)
	if err != nil {
		return "", err
	}
	var config UpgradeConfig
	err = json.Unmarshal(src, &config)
	if err != nil {
		return "", err
	}
	osArch := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	uri := config.Binaries[osArch]
	if uri == "" {
		return "", fmt.Errorf("cannot find binary for os/arch: %s", osArch)
	}
	return bootstrapBinary(path.Join("upgrades", upgradeName), uri, false)
}

func getDataDir() string {
	daemonHome, found := os.LookupEnv("DAEMON_HOME")
	if !found {
		panic("DAEMON_HOME environment variable must be set")
	}
	return filepath.Join(daemonHome, "upgrade_manager")
}

func bootstrapBinary(upgradeName string, uri string, force bool) (string, error) {
	path := filepath.Join(getDataDir(), url.PathEscape(upgradeName))
	_, err := os.Lstat(path)
	if err != nil || force {
		err := getter.GetFile(path, uri)
		if err != nil {
			return "", err
		}
	}
	return path, nil
}

func getCurLink() string {
	return filepath.Join(getDataDir(), "current")
}

func symlinkCurrentBinary(path string) {
	os.Remove(getCurLink())
	err := os.Symlink(path, getCurLink())
	if err != nil {
		panic(err)
	}
}

func (p *proc) launchProc(path string) {
	p.upgrading = true
	existing := p.cmd
	p.cmd = nil
	if existing != nil {
		_ = existing.Process.Kill()
	}
	cmd := exec.Command(path, p.args...)
	p.cmd = cmd
	cmd.Stdout = upgradeListener{p, os.Stdout}
	cmd.Stderr = upgradeListener{p, os.Stderr}
	err := cmd.Start()
	if err != nil {
		panic(err)
	}
	symlinkCurrentBinary(path)
	p.upgrading = false
	go func() {
		err = cmd.Wait()
		if !p.upgrading {
			if err == nil {
				os.Exit(0)
			} else {
				os.Exit(-1)
			}
		}
	}()
}

func main() {
	var path string
	// first check if there is an existing binary symlinked to current
	fi, err := os.Lstat(getCurLink())
	if err == nil && fi.Mode()&os.ModeSymlink != 0 {
		path, err = os.Readlink(getCurLink())
		if err != nil {
			panic(err)
		}
	} else {
		// then check if there is a binary setup at genesis
		path, err = bootstrapBinary("genesis", "", false)
		if err != nil {
			// now try checking if there is a binary setup in GENESIS_BINARY
			genBin := os.Getenv("GENESIS_BINARY")
			if genBin == "" {
				panic(fmt.Errorf("no binary configured, please use the GENESIS_BINARY environment variable to setup a genesis binary or setup one at %s/genesis", getDataDir()))
			}
			path, err = bootstrapBinary("genesis", genBin, true)
			if err != nil {
				panic(err)
			}
		}
	}
	p := &proc{args: os.Args[1:]}
	go p.launchProc(path)
	select {}
}
