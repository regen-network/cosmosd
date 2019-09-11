package main

import (
	"bufio"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

// LaunchProcess runs a subprocess and returns when the subprocess exits,
// either when it dies, or *after* a successful upgrade.
func LaunchProcess(cfg *Config, args []string, stdout, stderr io.Writer) error {
	bin := cfg.CurrentBin()
	err := EnsureBinary(bin)
	if err != nil {
		return errors.Wrap(err, "current binary invalid")
	}

	cmd := exec.Command(bin, args...)
	outpipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	errpipe, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	scanOut := bufio.NewScanner(io.TeeReader(outpipe, stdout))
	scanErr := bufio.NewScanner(io.TeeReader(errpipe, stderr))

	err = cmd.Start()
	if err != nil {
		return errors.Wrapf(err, "launching process %s %s", bin, strings.Join(args, " "))
	}

	// three ways to exit - command ends, find regexp in scanOut, find regexp in scanErr
	upgradeInfo, err := WaitForUpgradeOrExit(cmd, scanOut, scanErr)
	if err != nil {
		return err
	}
	if upgradeInfo != nil {
		return DoUpgrade(cfg, upgradeInfo)
	}

	return nil
}

// WaitForUpgradeOrExit listens to both output streams of the process, as well as the process state itself
// When it returns, the process is finished and all streams have closed.
//
// It returns (info, nil) if an upgrade should be initiated (and we killed the process)
// It returns (nil, err) if the process died by itself, or there was an issue reading the pipes
// It returns (nil, nil) if the process exited normally without triggering an upgrade. This is very unlikely
// to happend with "start" but may happend with short-lived commands like `gaiad export ...`
func WaitForUpgradeOrExit(cmd *exec.Cmd, scanOut, scanErr *bufio.Scanner) (*UpgradeInfo, error) {
	wg := sync.WaitGroup{}
	var err error
	var info *UpgradeInfo
	var mutex sync.Mutex

	// setError will set with the first error using a mutex
	// don't set it once info is set, that means we chose to kill the process
	setError := func(myErr error) {
		mutex.Lock()
		if err == nil && info == nil && myErr != nil {
			err = myErr
		}
		mutex.Unlock()
	}

	// set to first non-nil upgrade info
	setUpgrade := func(up *UpgradeInfo) {
		mutex.Lock()
		if info == nil && up != nil {
			info = up
			// now we need to kill the process
			_ = cmd.Process.Kill()
		}
		mutex.Unlock()
	}

	waitScan := func(scan *bufio.Scanner) {
		upgrade, err := WaitForUpdate(scanOut)
		if err != nil {
			setError(err)
		} else if upgrade != nil {
			setUpgrade(upgrade)
		}
		wg.Done()
	}

	// wait for command to exit
	wg.Add(3)
	go func() {
		if err := cmd.Wait(); err != nil {
			setError(err)
		}
		wg.Done()
	}()

	// wait for the scanners
	go waitScan(scanOut)
	go waitScan(scanErr)

	// wait for everyone to finish....
	wg.Wait()
	return info, err
}
