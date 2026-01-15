package rpc

import (
	"fmt"
	"strings"

	"github.com/0xAF4/go-monero/levin"
	"github.com/0xAF4/go-monero/types"
)

type Client struct{}

func NewDaemonRPCClient() *Client {
	return &Client{}
}

func (c *Client) GetBlocks(heights []uint64) ([]*types.Block, error) {
	req := UniversalRequest{
		"heights": heights,
	}

	// Для /get_blocks_by_height.bin используем JSON в запросе
	response, err := c.cycleCall(cGetBlocks, req.MarshalToBlob())
	if err != nil {
		return nil, fmt.Errorf(cErrorTxtTemplate, 1, cGetBlocks, err)
	}

	resp := make(UniversalRequest)
	if err := resp.FromPortableStorate(response); err != nil {
		return nil, fmt.Errorf(cErrorTxtTemplate, 2, cGetBlocks, err)
	}

	if strings.ToLower(resp["status"].(string)) != "ok" {
		return nil, fmt.Errorf("error, request is not ok!")
	}

	var blocksArr []*types.Block
	for _, blk := range resp["blocks"].(levin.Entries) {
		block := types.NewBlock()
		for _, ibl := range blk.Entries() {
			if ibl.Name == "block" {
				block.SetBlockData([]byte(ibl.String()))
			}
			if ibl.Name == "txs" {
				for _, itx := range ibl.Entries() {
					block.InsertTx([]byte(itx.String()))
				}
			}
		}
		blocksArr = append(blocksArr, block)
	}

	for _, blk := range blocksArr {
		blk.FullfillBlockHeader()
	}

	return blocksArr, nil
}

func (c *Client) GetTransactions(txIds []string) (*UniversalRequest, error) {
	req := UniversalRequest{
		"txs_hashes":     txIds,
		"decode_as_json": false,
		"prunable":       false,
	}

	response, err := c.cycleCall(cGetTransaction, req.MarshalToJson())
	if err != nil {
		return nil, fmt.Errorf(cErrorTxtTemplate, 1, cGetBlocks, err)
	}

	resp := make(UniversalRequest)
	resp.FromJson(response)
	return
}
