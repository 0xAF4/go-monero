package rpc

import (
	"encoding/binary"
	"encoding/hex"
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

func (c *Client) GetTransactions(txIds []string) (*[]UniversalRequest, error) {
	req := UniversalRequest{
		"txs_hashes":     txIds,
		"decode_as_json": false,
		"prunable":       false,
	}

	response, err := c.cycleCall(cGetTransaction, req.MarshalToJson())
	if err != nil {
		return nil, fmt.Errorf(cErrorTxtTemplate, 1, cGetTransaction, err)
	}

	resp := make(UniversalRequest)
	resp.FromJson(response)

	if strings.ToLower(resp["status"].(string)) != "ok" {
		return nil, fmt.Errorf("error, request is not ok!")
	}

	var txs []UniversalRequest
	for _, val := range resp["txs"].([]interface{}) {
		vvv := val.(map[string]interface{})
		data, _ := hex.DecodeString(vvv["as_hex"].(string))
		hexTx := types.Transaction{
			Raw: data,
		}
		hexTx.ParseTx()
		hexTx.ParseRctSig()
		hexTx.CalcHash()

		tx := UniversalRequest{
			"hash":           vvv["tx_hash"],
			"output_indices": vvv["output_indices"],
			"block_height":   vvv["block_height"],
			"extra":          []byte(hexTx.Extra),
		}
		txs = append(txs, tx)
	}

	return &txs, nil
}

func (c *Client) GetOutputDistribution(currentBlockHeight uint64) ([]uint64, error) {
	req := UniversalRequest{
		"amounts":     []uint64{0},
		"from_height": currentBlockHeight - 10,
		"cumulative":  true,
	}

	response, err := c.cycleCall(cGetOutputDistribution, req.MarshalToBlob())
	if err != nil {
		return nil, fmt.Errorf(cErrorTxtTemplate, 1, cGetOutputDistribution, err)
	}

	resp := make(UniversalRequest)
	if err := resp.FromPortableStorate(response); err != nil {
		return nil, fmt.Errorf(cErrorTxtTemplate, 2, cGetOutputDistribution, err)
	}

	if strings.ToLower(resp["status"].(string)) != "ok" {
		return nil, fmt.Errorf("error, request is not ok!")
	}

	var distributions []uint64
	for _, distr := range resp["distributions"].(levin.Entries) {
		for _, k1 := range distr.Entries() {
			if k1.Name == "distribution" {
				var bytes []byte
				switch v := k1.Value.(type) {
				case []byte:
					bytes = v
				case string:
					bytes = []byte(v)
				}

				const UINT_SIZE = 8
				if len(bytes)%UINT_SIZE != 0 {
					return nil, fmt.Errorf("invalid distribution length: %d, expected multiple of %d", len(bytes), UINT_SIZE)
				}

				numHashes := len(bytes) / UINT_SIZE

				for i := 0; i < numHashes; i++ {
					start := i * UINT_SIZE
					end := start + UINT_SIZE
					obj := binary.LittleEndian.Uint64(bytes[start:end])
					// Конвертируем в hex строку
					distributions = append(distributions, obj)
				}

			}
		}
	}

	return distributions, nil
}
