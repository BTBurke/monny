package stat

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// randNorm returns a []float64 array of normally distributed numbers with mean and stddev
// optional transform function can be used to return log-normally distributed numbers
func randNorm(length int, mean float64, stddev float64, transform func(float64) float64) []float64 {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	out := make([]float64, length)
	for i := 0; i < length; i++ {
		switch transform {
		case nil:
			out[i] = r.NormFloat64()*stddev + mean
		default:
			out[i] = transform(r.NormFloat64()*stddev + mean)
		}
	}
	return out
}

// logNormalTransform will generate Log-Normally distributed random numbers when passed as the transform
// to randNorm
var logNormalTransform func(float64) float64 = math.Exp

func TestMean(t *testing.T) {
	assert.Equal(t, mean([]float64{1.0, 1.0, 1.0, 2.0, 2.0, 2.0}), 1.5)
}

func TestVariance(t *testing.T) {
	values := []float64{1.0, 1.0, 1.0, 2.0, 2.0, 2.0}
	assert.Equal(t, variance(values, 1.5), 0.3)
}

func TestLimitCalc(t *testing.T) {
	values := []float64{1.0, 1.0, 1.0, 2.0, 2.0, 2.0}
	tt := []struct {
		name        string
		exp         float64
		sensitivity float64
		direction   int
	}{
		{name: "ucl", exp: 2.12105, sensitivity: 0.0, direction: 1},
		{name: "lcl", exp: 0.87894, sensitivity: 0.0, direction: -1},
		{name: "lcl more sensitive", exp: 1.003152797, sensitivity: 0.2, direction: -1},
		{name: "ucl more sensitive", exp: 1.996847203, sensitivity: 0.2, direction: 1},
		{name: "ucl less sensitive", exp: 2.245270804, sensitivity: -0.2, direction: 1},
		{name: "lcl less sensitive", exp: 0.754729196, sensitivity: -0.2, direction: -1},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			assert.InDelta(t, tc.exp, calculateLimit(mean(values), variance(values, mean(values)), 0.25, 3.0, tc.sensitivity, tc.direction), 0.00001)
		})
	}
}
