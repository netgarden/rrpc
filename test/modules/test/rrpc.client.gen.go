package test

import (
    "io"
    "net/http"
    "strings"
)

var DataMethods = []string{"POST", "PUT"}

func NewClient(url string) *Client {

	if string(url[len(url)-1]) == "/" {
		url = url[:len(url)-1]
	}

	c := &Client{
		url:    url,
		client: &http.Client{},
	}

	c.Pinger = newPingerServiceClientHandler(c)
    c.Static = newStaticServiceClientHandler(c)
    
	return c
}

type Client struct {
	url    string
	client *http.Client

	Pinger *PingerServiceClientHandler
	Static *StaticServiceClientHandler
	
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
    return c.client.Do(req)
}

// toWebSocketURL swaps an "http"/"https" base URL for its "ws"/"wss"
// counterpart - used by generated streaming client methods to dial, since
// websocket.Dial requires a ws(s):// URL, not http(s)://.
func toWebSocketURL(httpURL string) string {
    if strings.HasPrefix(httpURL, "https://") {
        return "wss://" + strings.TrimPrefix(httpURL, "https://")
    }
    if strings.HasPrefix(httpURL, "http://") {
        return "ws://" + strings.TrimPrefix(httpURL, "http://")
    }
    return httpURL
}

func NewSizedReader(r io.Reader, size int64) *SizedReader {
    return &SizedReader{
        size: size,
        r: r,
    }
}

type SizedReader struct {
    size int64
    r io.Reader
}

func (r *SizedReader) Size() int64 {
    return r.size
}

func (r *SizedReader) Read(p []byte) (n int, err error) {
    return r.r.Read(p)
}

func NewSizedReadCloser(r io.ReadCloser, size int64) *SizedReadCloser {
    return &SizedReadCloser{
        SizedReader: SizedReader{
            size: size,
            r: r,
        },
        r: r,
    }
}

type SizedReadCloser struct {
    SizedReader
    r io.ReadCloser
}

func (r *SizedReadCloser) Close() {
    r.r.Close()
}