package main

import (
	"bufio"
	"regexp"
	"strconv"

	"github.com/pkg/errors"
)

// Trim off whitespace around the info - match least greedy, grab as much space on both sides
var upgradeRegex = regexp.MustCompile(`UPGRADE "(.*)" NEEDED at height (\d+):\s+([^\s]*)\s*.*$`)

// UpgradeInfo is the details from the regexp
type UpgradeInfo struct {
	Name   string
	Height int
	Info   string
}

// WaitForUpdate will listen to the scanner until a line matches upgradeRegexp.
// It returns (info, nil) on a matching line
// It returns (nil, err) if the input stream errored
// It returns (nil, nil) if the input closed without ever matching the regexp
func WaitForUpdate(scanner *bufio.Scanner) (*UpgradeInfo, error) {
	for scanner.Scan() {
		line := scanner.Text()
		if upgradeRegex.MatchString(line) {
			subs := upgradeRegex.FindStringSubmatch(line)
			h, err := strconv.Atoi(subs[2])
			if err != nil {
				return nil, errors.Wrap(err, "parse number from regexp")
			}
			info := UpgradeInfo{
				Name:   subs[1],
				Height: h,
				Info:   subs[3],
			}
			return &info, nil
		}
	}
	return nil, scanner.Err()
}
