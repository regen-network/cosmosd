package main

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
			w, scan := ScanningWriter(ioutil.Discard)

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
