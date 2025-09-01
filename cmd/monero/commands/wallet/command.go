package wallet

import (
	"github.com/spf13/cobra"

	"github.com/0xaf4/go-monero/cmd/monero/options"
)

var RootCommand = &cobra.Command{
	Use:   "wallet",
	Short: "execute remote procedure calls against a monero wallet rpc server",
}

func init() {
	options.Bind(RootCommand)
}
