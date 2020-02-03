package stat

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/BTBurke/monny/pkg/metric"
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

func TestMetric(t *testing.T) {
	n, _ := NewLogNormalTest(metric.NewName("test_latency", nil), WithLogNormalStatistic(DefaultLogNormalEWMA()))
	est := n.sub[0].(*TestStatistic)
	est.current = 3.2222
	est.limit = 4.1111
	exp := map[string]float64{
		"test_latency[strategy=ewma type=estimator value=current]": 3.2222,
		"test_latency[strategy=ewma type=estimator value=limit]":   4.1111,
	}
	out := n.Metric()
	assert.Equal(t, exp, out)
}

func TestEWMAEstimator(t *testing.T) {
	gen := func(length int, mean float64) []float64 {
		return randNorm(length, mean, 1.0, logNormalTransform)
	}
	series := make([]float64, 0)
	series = append(append(series, gen(100, 5.2983)...), gen(300, 5.7038)...)

	est, _ := NewLogNormalTest(metric.NewName("test", nil), WithLogNormalStatistic(DefaultLogNormalEWMA()))
	ewma := est.sub[0].(*TestStatistic)
	for i, s := range series {
		if err := ewma.Record(s); err != nil {
			t.Fail()
		}
		if i == 51 {
			assert.Equal(t, TestingUCL, ewma.State())
		}
	}
	assert.Equal(t, UCLTrip, ewma.State())
}

// Measures the average number of samples to detect shifts in the mean. Test cases are represented as an increase
// in the mean as a multiple of the standard deviation.
func BenchmarkEWMA(b *testing.B) {
	// mean shifts as a multiple of the standard deviation
	tt := []float64{3, 2.5, 2.0, 1.8, 1.6, 1.4, 1.2, 1.0, 0.8, 0.6, 0.4, 0.2, 0.1, 0.05}
	for _, tc := range tt {
		b.Run(fmt.Sprintf("%0.2fσ", tc), func(b *testing.B) {
			samps := 0
			for i := 0; i < b.N; i++ {
				mean := 5.2983
				stdev := 1.0

				r := rand.New(rand.NewSource(time.Now().UnixNano()))
				next := func() float64 {
					return math.Exp(r.NormFloat64()*stdev + (mean + tc*stdev))
				}

				initial := randNorm(100, mean, stdev, logNormalTransform)
				e, _ := NewLogNormalTest(metric.NewName("asn_benchmark", nil), WithLogNormalStatistic(DefaultLogNormalEWMA()))
				est := e.sub[0].(*TestStatistic)
				for _, obs := range initial {
					if err := est.Record(obs); err != nil {
						b.Fail()
					}
				}
				s := 0
				for est.State() != UCLTrip && s <= 10000 {
					s++
					if err := est.Record(next()); err != nil {
						b.Fail()
					}
				}
				samps += s
			}
			b.ReportMetric(float64(samps)/float64(b.N), "samples(avg)")
		})
	}
}

// Measures the average number of samples to detect shifts in the mean. Test cases are represented as an increase
// in the mean as a multiple of the standard deviation.
func BenchmarkShewart(b *testing.B) {
	// mean shifts as a multiple of the standard deviation
	tt := []float64{3, 2.5, 2.0, 1.8, 1.6, 1.4, 1.2, 1.0, 0.8, 0.6, 0.4, 0.2, 0.1, 0.05}
	for _, tc := range tt {
		b.Run(fmt.Sprintf("%0.2fσ", tc), func(b *testing.B) {
			samps := 0
			for i := 0; i < b.N; i++ {
				mean := 5.2983
				stdev := 1.0

				r := rand.New(rand.NewSource(time.Now().UnixNano()))
				next := func() float64 {
					return math.Exp(r.NormFloat64()*stdev + (mean + tc*stdev))
				}

				initial := randNorm(100, mean, stdev, logNormalTransform)
				e, _ := NewLogNormalTest(metric.NewName("asn_benchmark", nil), WithLogNormalStatistic(DefaultLogNormalShewart()))
				est := e.sub[0].(*TestStatistic)
				for _, obs := range initial {
					if err := est.Record(obs); err != nil {
						b.Fail()
					}
				}
				s := 0
				for est.State() != UCLTrip && s <= 10000 {
					s++
					if err := est.Record(next()); err != nil {
						b.Fail()
					}
				}
				samps += s
			}
			b.ReportMetric(float64(samps)/float64(b.N), "samples(avg)")
		})
	}
}
