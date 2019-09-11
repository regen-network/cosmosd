package main

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"

	"github.com/pkg/errors"
)

// Trim off whitespace around the info - match least greedy, grab as much space on both sides
var upgradeRegex = regexp.MustCompile(`UPGRADE "(.*)" NEEDED at height (\d+):\s+(.*?)\s*$`)

type splitWriter struct {
	multi   io.Writer
	closers []io.Closer
}

func (s *splitWriter) Write(p []byte) (int, error) {
	n, err := s.multi.Write(p)
	fmt.Printf("[WRITE] Got %d, wrote %d, err=%s\n", len(p), n, err)
	return n, err
}

func (s *splitWriter) Close() error {
	for _, c := range s.closers {
		err := c.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// ScanningWriter will wrap the input writer
// The output is a new writer that will both pass-through write to the original writer,
// as well as a scanner of (a copy of) all the content
func ScanningWriter(w io.Writer) (io.WriteCloser, *bufio.Scanner) {
	pr, pw := io.Pipe()
	multi := io.MultiWriter(w, pw)
	closers := []io.Closer{pw}
	if cl, ok := w.(io.Closer); ok {
		closers = append(closers, cl)
	}

	writer := &splitWriter{multi, closers}
	return writer, bufio.NewScanner(pr)
}

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
		fmt.Printf("<SCANNER> %s\n", line)
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
