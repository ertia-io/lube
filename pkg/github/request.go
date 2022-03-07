package github

import (
	"bytes"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

var transport = &http.Transport{
	DialContext: (&net.Dialer{
		// Maximum amount of time a dial will wait for
		// a TCP connect to complete.
		Timeout: 5 * time.Second,
	}).DialContext,

	// Time to wait for a TLS handshake.
	TLSHandshakeTimeout: 5 * time.Second,

	// Amount of time to wait for server response headers after
	// fully writing the request (including its body, if any).
	ResponseHeaderTimeout: 5 * time.Second,

	// URL of the proxy to use for a given request, as indicated by the environment
	// variables HTTP_PROXY, HTTPS_PROXY and NO_PROXY (or the lowercase versions thereof).
	Proxy: http.ProxyFromEnvironment,
}

func (g *Github) request(method, url string, data []byte, class int) (int, []byte, error) {
	client := http.Client{
		Transport: transport,

		// Total time limit for request.
		Timeout: 30 * time.Second,

		// As we do not want to follow redirect.
		//CheckRedirect: func(req *http.Request, via []*http.Request) error {
		//	return http.ErrUseLastResponse
		//},
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		return 0, nil, err
	}

	switch class {
	case applicationJson:
		req.Header.Set("Accept", "application/json")
		if method == http.MethodPost || method == http.MethodPut {
			req.Header.Set("Content-Type", "application/json")
		}
	case applicationOctetStream:
		req.Header.Set("Accept", "application/octet-stream")
	}

	if len(g.apiToken) > 0 {
		token := "token " + g.apiToken
		req.Header.Set("Authorization", token)
	}

	res, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return 0, nil, err
	}

	if g.debug {
		// TODO
	}

	return res.StatusCode, body, nil
}
