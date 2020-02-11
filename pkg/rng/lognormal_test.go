package rng

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalRNG(t *testing.T) {
	r := NewLogNormalRNG(5.0, 1.0)
	val := make([]float64, 10000)
	for i := 0; i < 10000; i++ {
		val[i] = r.Rand()
	}

	sum := 0.0
	for _, v := range val {
		sum += math.Log(v)
	}
	mean := sum / float64(10000)
	assert.InDelta(t, 5.0, mean, 0.05)

	variance := 0.0
	for _, v := range val {
		variance += math.Pow(math.Log(v)-mean, 2.0)
	}
	variance = variance / float64(10000-1)
	assert.InDelta(t, 1.0, math.Sqrt(variance), 0.05)
}
