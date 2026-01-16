package types

// RPCClient определяет интерфейс для взаимодействия с Monero daemon
type RPCClient interface {
	GetTransactions(txIds []string) (*[]map[string]interface{}, error)
	GetOutputDistribution(currentBlockHeight uint64) ([]uint64, error)
	GetOuts(indxs []uint64) ([]*map[string]interface{}, error)
}
