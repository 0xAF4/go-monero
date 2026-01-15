package test

import (
	"fmt"
	"testing"

	"github.com/0xAF4/go-monero/rpc"
)

func Test_DaemonRPC_GetBlocks(t *testing.T) {
	client := rpc.NewDaemonRPCClient()

	resp, err := client.GetBlocks([]uint64{3517762, 3527762, 3537762, 3547762, 3557762})
	if err != nil {
		t.Fatalf("GetBlocks returned error: %v", err)
	}

	if resp == nil {
		t.Fatal("response is nil")
	}

	for _, block := range resp {
		fmt.Printf("block: %s\n", block.GetBlockId())
	}
}

func Test_DaemonRPC_GetTransactions(t *testing.T) {
	client := rpc.NewDaemonRPCClient()

	resp, err := client.GetTransactions([]string{"5a0247682c4170b643150434198a04d73270b98dd4c112c852ee01efaec30c19"})
	if err != nil {
		t.Fatalf("GetBlocks returned error: %v", err)
	}

	if resp == nil {
		t.Fatal("response is nil")
	}

	for _, tx := range *resp {
		fmt.Printf("TxHash: %s\n", tx["hash"])
		fmt.Printf("Extra: %x\n", tx["extra"])
		fmt.Printf("BlockHeight: %d\n", uint64(tx["block_height"].(float64)))
		fmt.Println("====")
	}
}
