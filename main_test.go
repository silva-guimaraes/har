package har

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestRun(t *testing.T) {
    files := []string {
        "docs.mitmproxy.org.har",
        "google_redirect[25-01-13 12-24-03].har",
        "tomscii.sig7.se_Archive [25-01-11 15-33-47].har",
        "null.har",
    }

	for _, fileName := range files {

		file, err := os.Open(filepath.Join("testdata", fileName))
		if err != nil {
			t.Fatal(err)
		}

		har, err := ReadHar(file)
		if err != nil {
			if errors.Is(err, ErrInvalidHar) && fileName == "null.har" {
				// expected
				continue
			} else {
				t.Fatal(err)
			}
		}

		t.Run(fileName, func(t *testing.T) {
			t.Parallel()
			for _, entry := range har.Entries() {
				resp, err := entry.DoRequest(nil)
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


func testPostData(t *testing.T, h *Har) {
    for _, entry := range h.Entries() {
        if entry.Request.Method != "POST" {
            continue
        }
        if entry.Request.PostData.MimeType == "" {
            t.Log(entry.Url())
            t.Errorf(`entry.PostData.MimeType == ""`)
        }
        if len(entry.Request.PostData.Params) == 0 && len(entry.Request.PostData.Text) == 0 {
            t.Log(entry.Url())
            t.Errorf(`len(entry.PostData.Params) == 0 && len(entry.PostData.Text) == 0`)
        }
    }
}

func TestPostData (t *testing.T) {
    file, err := os.Open(filepath.Join("testdata", "1post.har"))
    if err != nil {
        t.Fatal(err)
    }
    h, err := ReadHar(file)
    if err != nil {
        t.Fatal(err)
    }
    testPostData(t, h)

    file, err = os.Open(filepath.Join("testdata", "funcionario[25-01-13 10-41-40].har"))
    if err != nil {
        t.Fatal(err)
    }
    h, err = ReadHar(file)
    if err != nil {
        t.Fatal(err)
    }
    testPostData(t, h)

    file, err = os.Open(filepath.Join("testdata", "funcionario_chrome.har"))
    if err != nil {
        t.Fatal(err)
    }
    h, err = ReadHar(file)
    if err != nil {
        t.Fatal(err)
    }
    testPostData(t, h)
}
