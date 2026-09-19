package imaging

import (
	"fmt"
	"image"
	"strconv"
	"strings"

	"github.com/corona10/goimagehash"
)

const AlgorithmVersion = "perceptual-hash-v1"

type Hashes struct {
	PHash uint64
	DHash uint64
	AHash uint64
}

type Comparison struct {
	PHashDistance int
	DHashDistance int
	AHashDistance int
	Score         float64
}

func ComputeHashes(imageData image.Image) (Hashes, error) {
	phash, err := goimagehash.PerceptionHash(imageData)
	if err != nil {
		return Hashes{}, err
	}
	dhash, err := goimagehash.DifferenceHash(imageData)
	if err != nil {
		return Hashes{}, err
	}
	ahash, err := goimagehash.AverageHash(imageData)
	if err != nil {
		return Hashes{}, err
	}
	return Hashes{PHash: phash.GetHash(), DHash: dhash.GetHash(), AHash: ahash.GetHash()}, nil
}

func Compare(source, candidate Hashes) Comparison {
	p := hammingDistance(source.PHash, candidate.PHash)
	d := hammingDistance(source.DHash, candidate.DHash)
	a := hammingDistance(source.AHash, candidate.AHash)
	score := 0.60*similarity(p) + 0.25*similarity(d) + 0.15*similarity(a)
	return Comparison{PHashDistance: p, DHashDistance: d, AHashDistance: a, Score: score}
}

func FormatHash(value uint64) string {
	return fmt.Sprintf("%016x", value)
}

func ParseHashes(phash, dhash, ahash string) (Hashes, error) {
	values := []string{phash, dhash, ahash}
	parsed := make([]uint64, len(values))
	for index, value := range values {
		var err error
		parsed[index], err = strconv.ParseUint(strings.TrimSpace(value), 16, 64)
		if err != nil {
			return Hashes{}, fmt.Errorf("解析感知哈希失败: %w", err)
		}
	}
	return Hashes{PHash: parsed[0], DHash: parsed[1], AHash: parsed[2]}, nil
}

func hammingDistance(left, right uint64) int {
	value := left ^ right
	count := 0
	for value != 0 {
		value &= value - 1
		count++
	}
	return count
}

func similarity(distance int) float64 {
	return 1 - float64(distance)/64
}
