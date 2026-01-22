package rpc

import (
	"bytes"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"
)

func (c *Client) getRandomDaemonNode() string {
	if c.hostList != nil && len(*c.hostList) > 0 {
		return (*c.hostList)[rand.IntN(len(*c.hostList))]
	}
	return cRPCDaemonNodes[rand.IntN(len(cRPCDaemonNodes))]
}

func (c *Client) cycleCall(method string, data []byte) ([]byte, error) {
	var (
		response []byte
		err      error
	)

	for try := 0; try < c.retriesCount; try++ {
		response, err = c.call(method, data)
		if err == nil {
			return response, nil
		}
		time.Sleep(time.Millisecond * 100)
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", c.retriesCount, err)
}

func (c *Client) call(method string, data []byte) ([]byte, error) {
	url := c.getRandomDaemonNode() + method

	contentType := "application/json"
	if strings.HasSuffix(method, ".bin") {
		contentType = "application/octet-stream"
	}

	httpClient := &http.Client{
		Timeout: c.timeout,
	}

	resp, err := httpClient.Post(url, contentType, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("http post to %s failed: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"response status %d from %s, body: %s",
			resp.StatusCode, url, string(body),
		)
	}

	return body, nil
}
