package types

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
)

const (
	BlocksPerDay = 720.0
	RecentDays   = 1.8
	RecentRatio  = 0.5
	GammaShape   = 19.28
	GammaScale   = 1.0 / 1.61
	MaxGammaDays = 365.0 * 10
)

func sampleGamma(r *rand.Rand, k, theta float64) float64 {
	// Marsaglia and Tsang
	if k < 1 {
		return sampleGamma(r, k+1, theta) * math.Pow(r.Float64(), 1.0/k)
	}

	d := k - 1.0/3.0
	c := 1.0 / math.Sqrt(9*d)

	for {
		x := r.NormFloat64()
		v := 1 + c*x
		if v <= 0 {
			continue
		}
		v = v * v * v
		u := r.Float64()

		if u < 1-0.0331*(x*x)*(x*x) {
			return d * v * theta
		}
		if math.Log(u) < 0.5*x*x+d*(1-v+math.Log(v)) {
			return d * v * theta
		}
	}
}

func sampleOutputAgeDays(r *rand.Rand) float64 {
	if r.Float64() < RecentRatio {
		return r.Float64() * RecentDays
	}

	age := sampleGamma(r, GammaShape, GammaScale)
	if age > MaxGammaDays {
		return MaxGammaDays
	}
	return age
}

func getOutputIndex(rpcClient RPCClient, txId string, vout int) (uint64, error) {
	if vout < 0 {
		return 0, fmt.Errorf("vout must be non-negative")
	}

	resp985, err := rpcClient.GetTransactions([]string{txId})
	if err != nil {
		return 0, fmt.Errorf("invalid output index value: %w", err)
	}

	indexUint64 := (*resp985)[0]["output_indices"].([]uint64)[vout]
	return indexUint64, nil
}

func BuildKeyOffsets(indices []uint64) ([]uint64, error) {
	if len(indices) == 0 {
		return nil, errors.New("empty indices")
	}

	// 1. Копируем и сортируем
	sorted := make([]uint64, len(indices))
	copy(sorted, indices)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	// 2. Проверка на дубликаты (важно)
	for i := 1; i < len(sorted); i++ {
		if sorted[i] == sorted[i-1] {
			return nil, errors.New("duplicate global index")
		}
	}

	// 3. Строим offsets
	offsets := make([]uint64, len(sorted))
	offsets[0] = sorted[0]

	for i := 1; i < len(sorted); i++ {
		offsets[i] = sorted[i] - sorted[i-1]
	}

	return offsets, nil
}

func getMaxGlobalIndex(rpcClient RPCClient, currentBlockHeight uint64) (uint64, error) {
	distrib, err := rpcClient.GetOutputDistribution(currentBlockHeight)
	if err != nil {
		return 0, err
	}

	return distrib[len(distrib)-1] - 1, nil
}

func GetMixins(rpcClient RPCClient, keyOffsets []uint64, inputIndx uint64) (*[]Mixin, *int, error) {
	indxs := append([]uint64(nil), keyOffsets...)
	for i := 1; i < len(indxs); i++ {
		indxs[i] = indxs[i] + indxs[i-1]
	}

	// заполняем outputs
	var OrderIndx int
	for i, idx := range indxs {
		if idx == inputIndx {
			OrderIndx = i
		}
	}

	dests, err := rpcClient.GetOuts(indxs)
	if err != nil {
		return nil, nil, err
	}

	mixins := new([]Mixin)
	for _, out := range dests {
		tout := *out
		dest, _ := hex.DecodeString(tout["key"].(string))
		mask, _ := hex.DecodeString(tout["mask"].(string))

		*mixins = append(*mixins, Mixin{
			Dest: Hash(dest),
			Mask: Hash(mask),
		})
	}

	return mixins, &OrderIndx, nil
}
