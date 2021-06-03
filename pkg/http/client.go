package http

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
)

type Client struct {
	cli *http.Client
}

type options struct {
	timeout         time.Duration
	maxConnsPerHost int
}

// Option configures how we set up the client.
type Option interface {
	apply(*options)
}

// funcOption wraps a function that modifies Options into an
// implementation of the Option interface.
type funcOption struct {
	f func(*options)
}

func (fdo *funcOption) apply(do *options) {
	fdo.f(do)
}

func newFuncOption(f func(*options)) *funcOption {
	return &funcOption{
		f: f,
	}
}

// WithTimeout returns a Option that configures a timeout for dialing a ClientConn initially.
func WithTimeout(timeout time.Duration) Option {
	return newFuncOption(func(o *options) {
		o.timeout = timeout
	})
}

// WithMaxConnPerHost returns a Option that configures a maxConnsPerHost for dialing a ClientConn initially.
func WithMaxConnPerHost(max int) Option {
	return newFuncOption(func(o *options) {
		o.maxConnsPerHost = max
	})
}

func NewClient(optFunc ...Option) *Client {
	opts := &options{}
	for _, f := range optFunc {
		f.apply(opts)
	}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   6 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   6,
		MaxConnsPerHost:       opts.maxConnsPerHost,
	}
	return &Client{cli: &http.Client{
		Timeout:   opts.timeout,
		Transport: transport,
	}}
}

// Get http get
func (c *Client) Get(url string, respBody interface{}) (header http.Header, err error) {
	header, err = c.Do("GET", url, nil, respBody)
	return
}

// Put http put
func (c *Client) Put(url string, reqBody interface{}, respBody interface{}) (err error) {
	_, err = c.Do("PUT", url, reqBody, respBody)
	return
}

// Post http post
func (c *Client) Post(url string, reqBody interface{}, respBody interface{}) (err error) {
	_, err = c.Do("POST", url, reqBody, respBody)
	return
}

// Do http do
func (c *Client) Do(method string, url string, reqBody interface{}, respBody interface{}) (header http.Header, err error) {
	var (
		resp    *http.Response
		content []byte
		body    io.Reader
		req     *http.Request
	)
	if reqBody != nil {
		content, err = json.Marshal(reqBody)
		if err != nil {
			return
		}
		body = bytes.NewReader(content)
	}
	req, err = http.NewRequest(method, url, body)
	if err != nil {
		return
	}
	if body != nil {
		req.Header.Add("Content-Type", "application/json")
	}
	resp, err = c.cli.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	header = resp.Header
	if resp.StatusCode != http.StatusOK {
		content, _ = ioutil.ReadAll(resp.Body)
		err = errors.Newf(resp.StatusCode, errors.UnknownReason, string(content))
		return
	}
	content, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if respBody != nil {
		err = json.Unmarshal(content, respBody)
		if err != nil {
			return
		}
	}
	return
}
