package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
)

type proc struct {
	args []string
	cmd *exec.Cmd
}

var upgradeRegex = regexp.MustCompile(`UPGRADE "(.*)" NEEDED at height (\d+):(.*)`)

type upgradeListener struct {
	proc *proc
	writer io.Writer
}

func loadConfig() Config {
	var config Config
	configPath := os.Getenv("COSMOS_UPGRADE_MANAGER_CONFIG")
	if configPath == "" {
		configPath = "upgrade_manager.json"
	}
	configSrc, err := ioutil.ReadFile(configPath)
	if err != nil {
		panic(err)
	}
	err = cdc.UnmarshalJSON(configSrc, &config)
	if err != nil {
		panic(err)
	}
	return config
}

type OnChainConfig struct {
	Resolver Resolver `json:"upgrade_resolver"`
}

func (listener upgradeListener) Write(p []byte) (n int, err error) {
	matches := upgradeRegex.FindAll(p, 0)
	if matches != nil {
		if len(matches) != 3 {
			panic(fmt.Errorf("unexpected upgrade string: %s", p))
		}
		name := string(matches[0])
		config := loadConfig()
		var resolver Resolver
		resolver, found := config.Upgrades[name]
		if !found {
			var onchainConfig OnChainConfig
			err = cdc.UnmarshalJSON(matches[2], &onchainConfig)
			if err != nil {
				panic(err)
			}
			resolver = onchainConfig.Resolver
		}
		listener.proc.LaunchProc(resolver)
	}
	return listener.writer.Write(p)
}

func (p *proc) LaunchProc(resolver Resolver) {
	existing := p.cmd
	p.cmd = nil
	if existing != nil {
		_ = existing.Process.Kill()
	}
	path := resolver.BinaryPath()
	expectedHash, err := hex.DecodeString(resolver.Sha256Hash())
	if err != nil {
		panic(err)
	}
	hasher := sha256.New()
	s, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	hasher.Write(s)
	hash := hasher.Sum(nil)
	if !bytes.Equal(hash, expectedHash) {
		panic(fmt.Errorf("binary %s has hash %x, expected %x", path, hash, expectedHash))
	}
	cmd := exec.Command(path, p.args...)
	p.cmd = cmd
	cmd.Stdout = upgradeListener{p, os.Stdout}
	cmd.Stderr = upgradeListener{p, os.Stderr}
	err = cmd.Start()
	if err != nil {
		panic(err)
	}
}

func main() {
	config := loadConfig()
	p := proc{os.Args, nil}
	p.LaunchProc(config.Genesis)
}
