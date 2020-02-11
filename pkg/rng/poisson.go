package rng

import (
	"math"
	"math/rand"
	"time"
)

var _ RNG = &PoissonRNG{}

// PoissonRNG generates Poisson distributed numbers using Knuth's algorithm
type PoissonRNG struct {
	lambda float64
	r      *rand.Rand
}

func (r *PoissonRNG) Rand() float64 {
	// Knuth's algorithm
	L := math.Pow(math.E, -r.lambda)
	var k int64 = 0
	var p float64 = 1.0

	for p > L {
		k++
		p = p * r.r.Float64()
	}
	return float64(k - 1)
}

func NewPoissonRNG(lambda float64) *PoissonRNG {
	return &PoissonRNG{
		lambda: lambda,
		r:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}
