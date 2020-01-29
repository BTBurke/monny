package metric

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSeriesRecord(t *testing.T) {
	tt := []struct {
		name     string
		capacity int
		obs      []float64
		exp      []float64
	}{
		{name: "underfill", capacity: 5, obs: []float64{1, 2, 3}, exp: []float64{1, 2, 3, 0, 0}},
		{name: "fill", capacity: 5, obs: []float64{1, 2, 3, 4, 5}, exp: []float64{1, 2, 3, 4, 5}},
		{name: "overfill", capacity: 3, obs: []float64{1, 2, 3, 4, 5}, exp: []float64{3, 4, 5}},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := NewSeries(tc.capacity)
			for _, o := range tc.obs {
				s.Record(o)
			}
			assert.Equal(t, tc.exp, s.Values())
		})
	}
}

func TestWithValues(t *testing.T) {
	s, err := NewSeries(6, WithValues([]float64{1, 2, 3, 4}))
	assert.NoError(t, err)
	assert.Equal(t, []float64{1, 2, 3, 4, 0, 0}, s.Values())
}
