package har

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

var (
	ErrInvalidHar        = fmt.Errorf("invalid .har file")
	ErrUnexpectedStatus  = fmt.Errorf("unexpected status code")
	ErrIncompleteRequest = fmt.Errorf("imcomplete request")
)

type header struct {
	Name, Value string
}

type entry struct {
	ResourceType string `json:"_resourceType"`
	Request      struct {
		Method  string
		Url     string
		Headers []header
	}
	Response struct {
		Status  int
		Headers []header
		Error   string `json:"_error"`
	}
	Cookies []struct {
		Name     string
		Value    string
		Path     string
		Domain   string
		Expires  time.Time
		HttpOnly bool
		Secure   bool
	}
	PostData struct {
		// Algumas requisições aparentam não informar o Content-Type entre os headers.
		// Isso deveria ser incluído como o Content-Type?
		MimeType string
		Text     []byte
	}
}

type Har struct {
	Log struct {
		Version string
		Entries []entry
	}
}

func (h *Har) Entries() []entry {
	return h.Log.Entries
}

func (e entry) Url() string {
	return e.Request.Url
}

func ReadHar(reader io.Reader) (*Har, error) {

	data := new(Har)

	bytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(bytes, data)
	if err != nil {
		return nil, err
	}

	if data.Log.Version == "" {
		return nil, ErrInvalidHar
	}

	return data, nil
}

func (entry entry) BuildRequest() (*http.Request, error) {

	if entry.Response.Status == 0 || entry.Response.Error != "" {
		return nil, ErrIncompleteRequest
	}

	var body io.Reader = nil
	if len(entry.PostData.Text) != 0 {
		body = bytes.NewReader(entry.PostData.Text)
	}

	req, err := http.NewRequest(entry.Request.Method, entry.Request.Url, body)
	if err != nil {
		return nil, err
	}

	for _, h := range entry.Request.Headers {
		switch h.Name {
		case ":method":
			fallthrough
		case ":path":
			fallthrough
		case ":authority":
			fallthrough
		case ":scheme":
		// do nothing
		default:
			req.Header.Set(h.Name, h.Value)
		}
	}
	if entry.PostData.MimeType != "" {
		req.Header.Set("Contety-Type", entry.PostData.MimeType)
	}

	for _, cookie := range entry.Cookies {
		c := &http.Cookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Secure:   cookie.Secure,
			HttpOnly: cookie.HttpOnly,
			Domain:   cookie.Domain,
			Expires:  cookie.Expires,
		}
		req.AddCookie(c)
	}

	return req, nil
}

var noRedirect = fmt.Errorf("no redirects")

func (entry entry) DoRequest() (*http.Response, error) {
	req, err := entry.BuildRequest()
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return noRedirect
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, noRedirect) {
			return resp, nil
		} else {
			return nil, err
		}
	}
	return resp, nil
}
