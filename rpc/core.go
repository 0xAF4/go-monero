package rpc

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"
)

func getRandomDaemonNode() string {
	rand.Seed(time.Now().UnixNano())
	return cRPCDaemonNodes[rand.Intn(len(cRPCDaemonNodes))]
}

func (c *Client) cycleCall(method string, data []byte) ([]byte, error) {
	var (
		response []byte
		try      int = 0
		err      error
	)

	for try < cRetriesCount {
		response, err = c.call(method, data)
		if err == nil {
			break
		}
	}

	if err != nil {
		return nil, err
	}

	return response, nil
}

func (c *Client) call(method string, data []byte) ([]byte, error) {
	resp, err := http.Post(getRandomDaemonNode()+method, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Response is %d, body: %s", resp.StatusCode, string(body))
	}

	return body, nil
}
