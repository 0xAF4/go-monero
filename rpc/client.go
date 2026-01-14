package rpc

import (
	"encoding/json"
	"fmt"

	"github.com/0xAF4/go-monero/levin"
)

type UniversalRequest map[string]interface{}

type Client struct {
}

func NewDaemonRPCClient() *Client {
	return &Client{}
}

func (c *Client) GetBlocks(heights []uint64) (*UniversalRequest, error) {
	req := UniversalRequest{
		"heights": heights,
	}

	response, err := c.cycleCall(cGetBlocks, req.MarshalToBlob())
	if err != nil {
		return nil, fmt.Errorf(cErrorTxtTemplate, 1, cGetBlocks, err)
	}

	rStorage, err := levin.NewPortableStorageFromBytes(response)
	if err != nil {
		return nil, fmt.Errorf(cErrorTxtTemplate, 2, cGetBlocks, err)
	}

	resp := make(UniversalRequest)
	resp.FromPortableStorate(*rStorage)

	return &resp, nil
}

func (u UniversalRequest) MarshalToJson() []byte {
	js, _ := json.Marshal(u)
	return js
}

func (u UniversalRequest) MarshalToBlob() []byte {
	pStorage := levin.PortableStorage{
		Entries: []levin.Entry{},
	}
	for key, val := range u {
		var sVal levin.Serializable
		switch v := val.(type) {
		case string:
			sVal = levin.BoostString(v)
		case uint64:
			sVal = levin.BoostUint64(v)
		case []uint64:
			sVal = levin.BoostTxIDs(v)
		}
		entry := levin.Entry{
			Name:         key,
			Serializable: sVal,
		}
		pStorage.Entries = append(pStorage.Entries, entry)
	}
	return pStorage.Bytes()
}

func (u *UniversalRequest) FromPortableStorate(rStorage levin.PortableStorage) {
	for _, val := range rStorage.Entries {
		(*u)[val.Name] = val.Value
	}
}
