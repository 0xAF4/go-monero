package test

import (
	"fmt"
	"testing"

	"github.com/0xAF4/go-monero/rpc"
	"github.com/0xAF4/go-monero/types"
)

const (
	Address        = "49AvioLCdkk5gSXww99nEKJV3tsyBxszEZeywa7K3jQi1qcBhz4AJhPU6sroCaEsqDMJg1iG5Sv1z78u6vsa1fQCRaXGb1w"
	PrivateViewKey = "4fd69daf111e62ad6d64bfa3a529751db91eb35ef547e00d58ca1a99aee98209"
	TxID           = "fafe9b40569b1d04b68967ab7576a4037a50ce6859a08f84f3815fcc82817f4f"
)

func Test_Transaction_CheckBalance(t *testing.T) {
	client := rpc.NewDaemonRPCClient(timeout)

	resp, err := client.GetTransactions([]string{TxID})
	if err != nil {
		t.Fatalf("GetTransactions returned error: %v", err)
	}

	if resp == nil {
		t.Fatal("response is nil")
	}
	data := (*resp)[0]["data"].([]byte)

	transaction := types.Transaction{Raw: data}
	transaction.ParseTx()
	transaction.ParseRctSig()
	transaction.CalcHash()

	funds, paymentID, err := transaction.CheckOutputs(Address, PrivateViewKey)
	if err != nil {
		fmt.Printf("  - TX checkOutputs error: %s", err)
	} else {
		fmt.Printf("  - TX checkOutputs find in tx: %.12f; PaymentID: %d", funds, paymentID)
	}

}
