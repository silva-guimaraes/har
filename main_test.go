package har

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestRun(t *testing.T) {
	files, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}

	for _, file := range files {

		if file.IsDir() {
			continue
		}

		file, err := os.Open(filepath.Join("testdata", file.Name()))
		if err != nil {
			t.Fatal(err)
		}

		har, err := ReadHar(file)
		if err != nil {
			if errors.Is(err, ErrInvalidHar) && file.Name() == "testdata/null.har" {
				// expected
				continue
			} else {
				t.Fatal(err)
			}
		}

		t.Run(file.Name(), func(t *testing.T) {
			t.Parallel()
			for _, entry := range har.Entries() {
				resp, err := entry.DoRequest()
				if err != nil {
					if errors.Is(err, ErrIncompleteRequest) {
						continue
					} else {
						t.Log(entry.Url())
						t.Error(err)
					}
				}
				if entry.Response.Status != resp.StatusCode {
					t.Log(entry.Url())
					err = fmt.Errorf(
						"%w: expected: '%d', returned: '%d'",
						ErrUnexpectedStatus, entry.Response.Status, resp.StatusCode,
					)
					t.Error(err)
				}
			}
		})
	}
}
