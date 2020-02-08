package metric

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type op func(s *SampledSeries)

func r(obs float64) op {
	return func(s *SampledSeries) {
		s.Record(obs)
	}
}

func d(delay string) op {
	return func(s *SampledSeries) {
		d, _ := time.ParseDuration(delay)
		time.Sleep(d)
	}
}

func TestSampledObservations(t *testing.T) {
	tt := []struct {
		name      string
		capacity  int
		window    time.Duration
		transform func([]float64) float64
		ops       []op
		exp       []float64
	}{
		{name: "average", capacity: 1, window: 100 * time.Millisecond, transform: SampleAverage, ops: []op{r(1.0), r(3.0), d("130ms")}, exp: []float64{2.0}},
		{name: "sum", capacity: 1, window: 100 * time.Millisecond, transform: SampleSum, ops: []op{r(1.0), r(3.0), d("130ms")}, exp: []float64{4.0}},
		{name: "max", capacity: 1, window: 100 * time.Millisecond, transform: SampleMax, ops: []op{r(1.0), r(3.0), d("130ms")}, exp: []float64{3.0}},
		{name: "min", capacity: 1, window: 100 * time.Millisecond, transform: SampleMin, ops: []op{r(1.0), r(3.0), d("130ms")}, exp: []float64{1.0}},
		{name: "overfill", capacity: 1, window: 100 * time.Millisecond, transform: SampleMin, ops: []op{r(1.0), r(3.0), d("230ms")}, exp: []float64{0.0}},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			s, done, err := NewSampledSeries(tc.capacity, tc.window, tc.transform)
			defer done()
			assert.NoError(t, err)
			for _, f := range tc.ops {
				f(s)
			}
			assert.Equal(t, tc.exp, s.Values())
		})
	}
}
