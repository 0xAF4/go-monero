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

	// Для /get_blocks_by_height.bin используем JSON в запросе
	response, err := c.cycleCall(cGetBlocks, req.MarshalToBlob())
	if err != nil {
		return nil, fmt.Errorf(cErrorTxtTemplate, 1, cGetBlocks, err)
	}

	fmt.Printf("response: %x\n", response)
	// Ответ приходит в бинарном формате (portable storage)
	rStorage, err := levin.NewPortableStorageFromBytes(response)
	if err != nil {
		return nil, fmt.Errorf(cErrorTxtTemplate, 2, cGetBlocks, err)
	}

	resp := make(UniversalRequest)
	resp.FromPortableStorate(*rStorage)

	return &resp, nil
}

func (u UniversalRequest) MarshalToJson() []byte {
	js, err := json.Marshal(u)
	if err != nil {
		panic(fmt.Errorf("failed to marshal json: %w", err))
	}
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
		default:
			panic(fmt.Errorf("unsupported type for key %s: %T", key, val))
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
