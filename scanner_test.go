package main

import (
	"bufio"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func ExampleScanner() {
	pr, pw := io.Pipe()
	scanner := bufio.NewScanner(pr)

	// write some data
	go func() {
		pw.Write([]byte("Genesis foo bar 1234\n"))
		time.Sleep(time.Second)
		pw.Write([]byte("UPGRADE \"chain2\" NEEDED at height 49: {}\n"))
		time.Sleep(time.Second)
		pw.Write([]byte("End Game\n"))
		pw.Close()
	}()

	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
	}
	// Genesis foo bar 1234
	// UPGRADE \"chain2\" NEEDED at height 49: {}
	// End Game
}

func TestWaitForInfo(t *testing.T) {
	cases := map[string]struct {
		write         []string
		expectUpgrade *UpgradeInfo
		expectErr     bool
	}{
		"no info": {
			write: []string{"some", "random\ninfo\n"},
		},
		"simple match": {
			write: []string{"first line\n", `UPGRADE "myname" NEEDED at height 123: http://example.com`, "\nnext line\n"},
			expectUpgrade: &UpgradeInfo{
				Name:   "myname",
				Height: 123,
				Info:   "http://example.com",
			},
		},
		"chunks": {
			write: []string{"first l", "ine\nERROR 2020-02-03T11:22:33Z: UPGRADE ", `"split" NEEDED at height `, "789:   {\"foo\": 123}", "  \n LOG: next line"},
			expectUpgrade: &UpgradeInfo{
				Name:   "split",
				Height: 789,
				Info:   `{"foo": 123}`,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r, w := io.Pipe()
			scan := bufio.NewScanner(r)

			// write all info in separate routine
			go func() {
				for _, line := range tc.write {
					n, err := w.Write([]byte(line))
					assert.NoError(t, err)
					assert.Equal(t, len(line), n)
				}
				w.Close()
			}()

			// now scan the info
			info, err := WaitForUpdate(scan)
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expectUpgrade, info)
		})
	}
}
