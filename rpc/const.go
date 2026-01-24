package rpc

var cRPCDaemonNodes = []string{
	"https://xmr.unshakled.net:443",
	"https://xmr1.doggett.tech:18089",
	"https://xmr3.doggett.tech:18089",
	"https://xmr5.doggett.tech:18089",
	"https://xmr-node.cakewallet.com:18081",
	"https://node.sethforprivacy.com:443",
	"https://nodes.hashvault.pro:18081",
	"https://xmr.cryptostorm.is:18081",
	"https://public-monero-node.xyz:443",
	"https://monero.openinternet.io:443",
	"https://node.xmr.surf:443",
	"https://xmr.kareem.one:443",
}

const (
	// cRetriesCount     = 3
	cErrorTxtTemplate = "Error(%d) of calling %s method: %w"
	cJSON_RPC         = "/json_rpc"

	cGetBlocks             = "/get_blocks_by_height.bin"
	cGetTransaction        = "/get_transactions"
	cGetOutputDistribution = "/get_output_distribution.bin"
	cGetOuts               = "/get_outs"
	cSendRawTransaction    = "/send_raw_transaction"
	cGetHeight             = "/get_height"
	cGetFeeEstimate        = "get_fee_estimate"
)
