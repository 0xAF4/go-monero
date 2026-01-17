package test

import (
	"fmt"
	"testing"
	"time"

	"github.com/0xAF4/go-monero/rpc"
)

const timeout = 10 * time.Second

func Test_DaemonRPC_GetBlocks(t *testing.T) {
	client := rpc.NewDaemonRPCClient(timeout)

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
	client := rpc.NewDaemonRPCClient(timeout)

	resp, err := client.GetTransactions([]string{"5a0247682c4170b643150434198a04d73270b98dd4c112c852ee01efaec30c19"})
	if err != nil {
		t.Fatalf("GetTransactions returned error: %v", err)
	}

	if resp == nil {
		t.Fatal("response is nil")
	}
	extra := (*resp)[0]["extra"]
	fmt.Println(extra)
	fmt.Printf("extra: %x\n", extra)

	for _, tx := range *resp {
		fmt.Printf("TxHash: %s\n", tx["hash"])
		fmt.Printf("Extra: %x\n", tx["extra"])
		fmt.Printf("BlockHeight: %d\n", uint64(tx["block_height"].(float64)))
		fmt.Println("====")
	}
}

func Test_DaemonRPC_GetOutputDistribution(t *testing.T) {
	client := rpc.NewDaemonRPCClient(timeout)

	resp, err := client.GetOutputDistribution(3557762)
	if err != nil {
		t.Fatalf("GetOutputDistribution returned error: %v", err)
	}

	if resp == nil {
		t.Fatal("response is nil")
	}

	fmt.Println("Distribution:", resp)
}

func Test_DaemonRPC_GetOuts(t *testing.T) { //TODO: to=do
	client := rpc.NewDaemonRPCClient(timeout)

	resp, err := client.GetOuts([]uint64{123456, 789012})
	if err != nil {
		t.Fatalf("GetOuts returned error: %v", err)
	}

	if resp == nil {
		t.Fatal("response is nil")
	}

	for _, out := range resp {
		fmt.Println(*out)
	}
}

func Test_DaemonRPC_SendRawTransaction(t *testing.T) {
	client := rpc.NewDaemonRPCClient(timeout)

	ok, err := client.SendRawTransaction("txHex", false)
	if err != nil {
		t.Fatalf("SendRawTransaction returned error: %v", err)
	}

	fmt.Println("Sended:", *ok)
}

func Test_DaemonRPC_GetFeeEstimate(t *testing.T) {
	client := rpc.NewDaemonRPCClient(timeout)

	fees, err := client.GetFeeEstimate()
	if err != nil {
		t.Fatalf("GetOutputDistribution returned error: %v", err)
	}

	fmt.Println("Fees:", *fees)
}

func Test_DaemonRPC_GetHeight(t *testing.T) {
	client := rpc.NewDaemonRPCClient(timeout)

	hash, height, err := client.GetHeight()
	if err != nil {
		t.Fatalf("GetHeight returned error: %v", err)
	}

	fmt.Println("hash:", hash)
	fmt.Println("height:", height)
}
