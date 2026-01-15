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

func getRandomDaemonNode() string {
	return cRPCDaemonNodes[rand.IntN(len(cRPCDaemonNodes))]
}

func (c *Client) cycleCall(method string, data []byte) ([]byte, error) {
	var (
		response []byte
		err      error
	)

	for try := 0; try < cRetriesCount; try++ {
		response, err = c.call(method, data)
		if err == nil {
			return response, nil
		}
		time.Sleep(time.Millisecond * 100)
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", cRetriesCount, err)
}

func (c *Client) call(method string, data []byte) ([]byte, error) {
	url := getRandomDaemonNode() + method

	contentType := "application/json"
	if strings.HasSuffix(method, ".bin") {
		contentType = "application/octet-stream"
	}

	resp, err := http.Post(url, contentType, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("http post to %s failed: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("response status %d from %s, body: %s",
			resp.StatusCode, url, string(body))
	}

	return body, nil
}
