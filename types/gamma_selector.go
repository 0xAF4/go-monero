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
	GammaScale   = 1.61 // ✅ БЕЗ деления на 1.61!
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
		// 50% шанс выбрать из недавних (последние 1.8 дня)
		return r.Float64() * RecentDays
	}

	// Используем gamma distribution для остальных
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

	// Копируем и сортируем (на всякий случай, если ещё не отсортировано)
	sorted := make([]uint64, len(indices))
	copy(sorted, indices)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	// Проверка на дубликаты
	for i := 1; i < len(sorted); i++ {
		if sorted[i] == sorted[i-1] {
			return nil, fmt.Errorf("duplicate global index at position %d: %d", i, sorted[i])
		}
	}

	// Строим offsets
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

func SelectDecoys(rng *rand.Rand, realGlobalIndex uint64, maxGlobalIndex uint64) ([]uint64, error) {
	const ringSize = 16

	if maxGlobalIndex < ringSize {
		return nil, fmt.Errorf("not enough outputs: max=%d, need=%d", maxGlobalIndex, ringSize)
	}

	selected := make(map[uint64]struct{})
	selected[realGlobalIndex] = struct{}{}

	maxAttempts := 10000
	attempts := 0

	for len(selected) < ringSize && attempts < maxAttempts {
		attempts++

		// 1. Генерируем возраст в днях используя gamma distribution
		ageDays := sampleOutputAgeDays(rng)

		// 2. Конвертируем возраст в блоки
		ageBlocks := uint64(ageDays * BlocksPerDay)

		// 3. ИСПРАВЛЕНО: правильно вычисляем глобальный индекс
		var gi uint64
		if ageBlocks >= maxGlobalIndex {
			// Слишком старый возраст - выбираем из начала диапазона
			gi = uint64(rng.Int63n(int64(maxGlobalIndex / 10)))
		} else {
			// Вычисляем базовый индекс: чем старше, тем меньше индекс
			baseIndex := maxGlobalIndex - ageBlocks

			// Добавляем небольшой случайный jitter для разнообразия
			jitterRange := int64(BlocksPerDay * 0.1) // ±10% от дня в блоках
			if jitterRange < 1 {
				jitterRange = 1
			}
			jitter := rng.Int63n(jitterRange*2) - jitterRange

			// Применяем jitter
			if jitter < 0 && baseIndex < uint64(-jitter) {
				gi = 0
			} else {
				gi = uint64(int64(baseIndex) + jitter)
			}

			// Проверяем границы
			if gi >= maxGlobalIndex {
				gi = maxGlobalIndex - 1
			}
		}

		// 4. Проверяем, что индекс уникален и не совпадает с реальным
		if gi == realGlobalIndex {
			continue
		}
		if _, exists := selected[gi]; exists {
			continue
		}

		selected[gi] = struct{}{}
	}

	if len(selected) < ringSize {
		return nil, fmt.Errorf("failed to select enough unique decoys: got %d after %d attempts",
			len(selected), attempts)
	}

	// 5. Конвертируем map в slice
	ring := make([]uint64, 0, ringSize)
	for gi := range selected {
		ring = append(ring, gi)
	}

	// 6. ✅ ОБЯЗАТЕЛЬНО СОРТИРУЕМ (не перемешиваем!)
	sort.Slice(ring, func(i, j int) bool {
		return ring[i] < ring[j]
	})

	return ring, nil
}
