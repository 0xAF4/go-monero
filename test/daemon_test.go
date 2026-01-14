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

	for key, val := range *resp {
		fmt.Println(key, ":", val)
	}
}
