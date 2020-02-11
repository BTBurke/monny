package rng

import (
	"math"
	"math/rand"
	"time"
)

var _ RNG = &LogNormalRNG{}

// LogNormalRNG generates Log Normal random numbers
type LogNormalRNG struct {
	mean  float64
	stdev float64
	r     *rand.Rand
}

func (r *LogNormalRNG) Rand() float64 {
	return math.Exp(r.r.NormFloat64()*r.stdev + r.mean)
}

func NewLogNormalRNG(mean float64, stdev float64) *LogNormalRNG {
	return &LogNormalRNG{
		mean:  mean,
		stdev: stdev,
		r:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}
