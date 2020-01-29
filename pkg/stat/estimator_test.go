package stat

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
		{name: "ucl", exp: 2.12105, sensitivity: 1.0, direction: 1},
		{name: "lcl", exp: 0.87894, sensitivity: 1.0, direction: -1},
		{name: "lcl sensitivity", exp: 0.73245, sensitivity: 1.2, direction: -1},
		{name: "ucl sensitivity", exp: 2.54527, sensitivity: 1.2, direction: 1},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			assert.InDelta(t, tc.exp, calculateLimit(mean(values), variance(values, mean(values)), 0.25, 3.0, tc.sensitivity, tc.direction), 0.00001)
		})
	}
}
