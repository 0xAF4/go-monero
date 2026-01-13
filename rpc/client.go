package rpc

import "encoding/json"

type UniversalRequest map[string]interface{}

type Client struct {
}

func NewDaemonRPCClient() *Client {
	return &Client{}
}

func (c *Client) GetBlocks(heights []uint64) (interface{}, error) {
	req := UniversalRequest{
		"heights": heights,
	}
	c.call(cGetBlocks, req.Marshal())
}

func (u UniversalRequest) Marshal() string {
	js, _ := json.Marshal(u)
	return string(js)
}
