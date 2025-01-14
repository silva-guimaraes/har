package har

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

var (
	ErrInvalidHar        = fmt.Errorf("invalid .har file")
	ErrUnexpectedStatus  = fmt.Errorf("unexpected status code")
	ErrIncompleteRequest = fmt.Errorf("imcomplete request")
)

type keyvalue struct {
	Name, Value string
}

type Entry struct {
	ResourceType string `json:"_resourceType"`
	Initiator    struct {
		Type string
	} `json:"_initiator"`
	Request struct {
		Method      string
		Url         string
		Headers     []keyvalue
		HeadersSize int `json:"headersSize"`
		BodySize    int `json:"bodySize"`
		PostData    struct {
			MimeType string `json:"mimeType"`
			Text     string
			Params   []keyvalue
		} `json:"postData"`
	}
	Response struct {
		Status  int
		Headers []keyvalue
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
	Time            float32
	StartedDateTime time.Time `json:"startedDateTime"`
	Timings         struct {
		Connect float32
	}
}

type Har struct {
	Log struct {
		Version string
		Entries []Entry
	}
}

func (h *Har) Entries() []Entry {
	return h.Log.Entries
}

func (e Entry) Url() string {
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

func (entry Entry) BuildRequest() (*http.Request, error) {

	if entry.Response.Status == 0 || entry.Response.Error != "" {
		return nil, ErrIncompleteRequest
	}

	var (
		request           = entry.Request
		body    io.Reader = nil
	)

	if len(request.PostData.Text) > 0 {
		body = strings.NewReader(request.PostData.Text)
	} else if len(request.PostData.Params) > 0 {
		values := url.Values{}
		for _, p := range request.PostData.Params {
			values.Add(p.Name, p.Value)
		}
		body = strings.NewReader(values.Encode())
	}

	req, err := http.NewRequest(request.Method, request.Url, body)
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
	if request.PostData.MimeType != "" {
		req.Header.Set("Contety-Type", request.PostData.MimeType)
	}

	// for _, cookie := range entry.Cookies {
	// 	c := &http.Cookie{
	// 		Name:     cookie.Name,
	// 		Value:    cookie.Value,
	// 		Secure:   cookie.Secure,
	// 		HttpOnly: cookie.HttpOnly,
	// 		Domain:   cookie.Domain,
	// 		Expires:  cookie.Expires,
	// 	}
	// 	req.AddCookie(c)
	// }

	return req, nil
}

var NoRedirect = fmt.Errorf("no redirects permitted")

func DefaultClient(options *cookiejar.Options) (*http.Client, error) {

	jar, err := cookiejar.New(options)
	if err != nil {
		return nil, err
	}

    client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return NoRedirect
		},
		Jar: jar,
	}
	return client, nil
}

func (entry Entry) DoRequest(client *http.Client) (*http.Response, error) {

	req, err := entry.BuildRequest()
	if err != nil {
		return nil, err
	}

	if client == nil {
        defaultClient, err := DefaultClient(nil)
        if err != nil {
            return nil, err
        }
        client = defaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, NoRedirect) {
			return resp, nil
		} else {
			return nil, err
		}
	}

	return resp, nil
}
