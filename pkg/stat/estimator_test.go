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

// randPoisson returns a []float64 array of poisson distributed values with the given lambda using
// Knuth's algorithm
func randPoisson(length int, lambda float64) []float64 {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	out := make([]float64, length)
	for i := 0; i < length; i++ {
		// algorithm by Knuth
		L := math.Pow(math.E, -lambda)
		var k int64 = 0
		var p float64 = 1.0

		for p > L {
			k++
			p = p * r.Float64()
		}
		out[i] = float64(k - 1)
	}
	return out
}

// logNormalTransform will generate Log-Normally distributed random numbers when passed as the transform
// to randNorm
var logNormalTransform func(float64) float64 = math.Exp

func TestMean(t *testing.T) {
	assert.Equal(t, meanNormal([]float64{1.0, 1.0, 1.0, 2.0, 2.0, 2.0}), 1.5)
	assert.Equal(t, meanPoisson([]float64{1.0, 1.0, 1.0, 2.0, 2.0, 2.0}), 1.5)
}

func TestVariance(t *testing.T) {
	values := []float64{1.0, 1.0, 1.0, 2.0, 2.0, 2.0}
	assert.Equal(t, varianceNormal(values, 1.5), 0.3)
	assert.Equal(t, variancePoisson(values, 1.5), 1.5)
}

func TestLimitCalc(t *testing.T) {
	values := []float64{1.0, 1.0, 1.0, 2.0, 2.0, 2.0}
	tt := []struct {
		name      string
		exp       float64
		direction int
	}{
		{name: "ucl", exp: 2.59064, direction: 1},
		{name: "lcl", exp: 0.40935, direction: -1},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			assert.InDelta(t, tc.exp, calculateLimit(meanNormal(values), varianceNormal(values, meanNormal(values)), 0.25, &KErrorRate{.05}, tc.direction), 0.00001)
		})
	}
}

func TestLNMetric(t *testing.T) {
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

func TestPMetric(t *testing.T) {
	n, _ := NewPoissonTest(metric.NewName("test_error_rate", nil), WithPoissonStatistic(DefaultPoissonEWMA()))
	defer n.Done()
	est := n.sub[0].(*TestStatistic)
	est.current = 3.2222
	est.limit = 4.1111
	exp := map[string]float64{
		"test_error_rate[strategy=ewma type=estimator value=current]": 3.2222,
		"test_error_rate[strategy=ewma type=estimator value=limit]":   4.1111,
	}
	out := n.Metric()
	assert.Equal(t, exp, out)
}

func TestLogNormalEWMAEstimator(t *testing.T) {
	gen := func(length int, mean float64) []float64 {
		return randNorm(length, mean, 1.0, logNormalTransform)
	}
	series := make([]float64, 0)
	series = append(append(series, gen(100, 5.2983)...), gen(2000, 8.0)...)

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

func TestPoissonEWMAEstimator(t *testing.T) {
	gen := func(length int, lambda float64) []float64 {
		return randPoisson(length, lambda)
	}
	series := make([]float64, 0)
	series = append(append(series, gen(100, 5)...), gen(600, 10)...)

	testStat, _ := NewEWMATestStatistic("ewma", 0.25, &KErrorRate{0.05}, NewPoisson(50, 10*time.Millisecond, metric.SampleMax))
	est, _ := NewPoissonTest(metric.NewName("test", nil), WithPoissonStatistic(testStat))
	ewma := est.sub[0].(*TestStatistic)
	for i, s := range series {
		if err := ewma.Record(s); err != nil {
			t.Fail()
		}
		time.Sleep(10 * time.Millisecond)
		if i == 51 {
			assert.Equal(t, TestingUCL, ewma.State())
		}
	}
	assert.Equal(t, UCLTrip, ewma.State())
}

// Measures the average number of samples to detect shifts in the mean of a log normal process. Test cases are represented as an increase
// in the mean as a multiple of the standard deviation.
func BenchmarkLogNormalEWMA(b *testing.B) {
	tt := []float64{3, 2.5, 2.0, 1.8, 1.6, 1.4, 1.2, 1.0}
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
				for est.State() != UCLTrip {
					s++
					if err := est.Record(next()); err != nil {
						b.Fail()
					}
				}
				samps += s
			}
			b.ReportMetric(0, "ns/op")
			b.ReportMetric(float64(samps)/float64(b.N), "samples(avg)")
		})
	}
}

// // Measures the average number of samples to detect shifts in the mean of a log normal process. Test cases are represented as an increase
// // in the mean as a multiple of the standard deviation.
// func BenchmarkLogNormalShewart(b *testing.B) {
//   // mean shifts as a multiple of the standard deviation
//   tt := []float64{3, 2.5, 2.0, 1.8, 1.6, 1.4, 1.2, 1.0, 0.8, 0.6, 0.4, 0.2, 0.1, 0.05}
//   for _, tc := range tt {
//     b.Run(fmt.Sprintf("%0.2fσ", tc), func(b *testing.B) {
//       samps := 0
//       for i := 0; i < b.N; i++ {
//         mean := 5.2983
//         stdev := 1.0
//
//         r := rand.New(rand.NewSource(time.Now().UnixNano()))
//         next := func() float64 {
//           return math.Exp(r.NormFloat64()*stdev + (mean + tc*stdev))
//         }
//
//         initial := randNorm(100, mean, stdev, logNormalTransform)
//         e, _ := NewLogNormalTest(metric.NewName("asn_benchmark", nil), WithLogNormalStatistic(DefaultLogNormalShewart()))
//         est := e.sub[0].(*TestStatistic)
//         for _, obs := range initial {
//           if err := est.Record(obs); err != nil {
//             b.Fail()
//           }
//         }
//         s := 0
//         for est.State() != UCLTrip && s <= 10000 {
//           s++
//           if err := est.Record(next()); err != nil {
//             b.Fail()
//           }
//         }
//         samps += s
//       }
//			b.ReportMetric(0, "ns/op")
//       b.ReportMetric(float64(samps)/float64(b.N), "samples(avg)")
//     })
//   }
// }
//
// Measures the average number of samples to detect shifts in the mean of a poisson process. Test cases are represented as an increase
// in the mean as a multiple of the standard deviation.
func BenchmarkPoissonEWMA(b *testing.B) {
	tt := []float64{3, 2.5, 2.0, 1.8, 1.6, 1.4, 1.2, 1.0}
	for _, tc := range tt {
		b.Run(fmt.Sprintf("%0.2fσ", tc), func(b *testing.B) {
			samps := 0
			for i := 0; i < b.N; i++ {
				lambda := 5.0
				next := func() float64 {
					r := randPoisson(1, lambda+math.Sqrt(tc)*lambda)
					return r[0]
				}

				initial := randPoisson(50, lambda)

				stat, _ := NewEWMATestStatistic("ewma", 0.25, &KErrorRate{.05}, NewPoisson(50, 0, nil))
				e, _ := NewPoissonTest(metric.NewName("asn_benchmark", nil), WithPoissonStatistic(stat))
				est := e.sub[0].(*TestStatistic)
				for _, obs := range initial {
					if err := est.Record(obs); err != nil {
						b.Fatalf("recording error: %v", err)
					}
				}
				if est.State() != TestingUCL {
					b.Fatalf("expected UCL testing, got %v", est.State())
				}
				s := 0
				for est.State() != UCLTrip {
					if err := est.Record(next()); err != nil {
						b.Fail()
					}
					s++
				}
				samps += s
			}
			b.ReportMetric(0, "ns/op")
			b.ReportMetric(float64(samps)/float64(b.N), "samples(avg)")
		})
	}
}

//
func BenchmarkLogNormalError(b *testing.B) {
	tt := []func() Test{
		func() Test {
			est, _ := NewLogNormalTest(metric.NewName("ewma", nil), WithLogNormalStatistic(DefaultLogNormalEWMA()))
			return est
		},
		func() Test {
			est, _ := NewLogNormalTest(metric.NewName("shewart", nil), WithLogNormalStatistic(DefaultLogNormalShewart()))
			return est
		},
	}
	for _, tc := range tt {
		est := tc()
		b.Run(est.Name(), func(b *testing.B) {
			avgError := 0.0
			for i := 0; i < b.N; i++ {
				errors := 0
				for j := 0; j < 1000; j++ {
					est := tc()
					values := randNorm(100000, 5.0, 1.0, logNormalTransform)
					for _, v := range values {
						est.Record(v)
						if est.HasAlarmed() {
							errors += 1
							break
						}
					}
				}
				avgError += float64(errors) / 1000.0
			}
			b.ReportMetric(0, "ns/op")
			b.ReportMetric(avgError/float64(b.N), "p(typeI)")
		})
	}
}

func BenchmarkPoissonError(b *testing.B) {
	tt := []func() Test{
		func() Test {
			stat, _ := NewEWMATestStatistic("ewma", 0.25, &KErrorRate{0.05}, NewPoisson(100, 0, metric.SampleSum))
			est, _ := NewPoissonTest(metric.NewName("ewma", nil), WithPoissonStatistic(stat))
			return est
		},
		func() Test {
			stat, _ := NewEWMATestStatistic("shewart", 1.0, &KErrorRate{0.05}, NewPoisson(100, 0, metric.SampleSum))
			est, _ := NewPoissonTest(metric.NewName("shewart", nil), WithPoissonStatistic(stat))
			return est
		},
	}
	for _, tc := range tt {
		b.Run(tc().Name(), func(b *testing.B) {
			avgError := 0.0
			for i := 0; i < b.N; i++ {
				errors := 0
				for j := 0; j < 1000; j++ {
					est := tc()
					values := randPoisson(100000, 50.0)
					for _, v := range values {
						est.Record(v)
						if est.HasAlarmed() {
							errors += 1
							break
						}
					}
				}
				avgError += float64(errors) / 1000.0
			}
			b.ReportMetric(0, "ns/op")
			b.ReportMetric(avgError/float64(b.N), "p(typeI)")
		})
	}
}
