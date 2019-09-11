package main

import (
	"bufio"
	"io"
	"regexp"
	"strconv"

	"github.com/pkg/errors"
)

var upgradeRegex = regexp.MustCompile(`UPGRADE "(.*)" NEEDED at height (\d+): (.*)$`)

type splitWriter struct {
	multi   io.Writer
	closers []io.Closer
}

func (s *splitWriter) Write(p []byte) (int, error) {
	return s.multi.Write(p)
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

// ScanningtWriter will wrap the input writer
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
