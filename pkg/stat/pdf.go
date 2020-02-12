package stat

import (
	"fmt"
	"math"
	"time"

	"github.com/BTBurke/monny/pkg/metric"
)

var _ PDF = &LogNormal{}
var _ PDF = &Poisson{}

// PDF is the assumed probability density function of the (possibly transformed) observations.  For Log-Normal, observations
// are first transformed as Log(obs), which is then normally distributed.  Other statistics may be better fit by a Poisson
// distribution, such as windowed counts of observations like 400/500 errors or server load (requests/time period)
type PDF interface {
	// Mean is a MLE of the (potentially transformed) distribution mean based on observed samples
	Mean(obs []float64) float64
	// Variance is the MLE of the distribution variance
	Variance(obs []float64, mean float64) float64
	// NewSeries returns a SeriesRecorder appropriate for the distribution type
	NewSeries() (metric.SeriesRecorder, error)
	// Transform will transform raw observations to the underlying tested distribution (e.g. LogNormal -> Normal)
	Transform(obs float64) float64
	// K returns the k value for upper and lower control limits based on the type of distribution and desired Type I error rate
	K() (float64, error)
	// Done is a cleanup function that tears down any running go routines necessary for maintaining series state
	Done()
	// String implements stringer
	String() string
}

// Poisson is a possion modeled process, such as request error rates, etc.  It would be useful for monitoring any metric
// in which the result is countable over a window, such as number of 400 responses for an API per minute, etc.
type Poisson struct {
	capacity int
	window   time.Duration
	strategy func([]float64) float64
	done     func()
	k        K
}

func (p *Poisson) Mean(obs []float64) float64 {
	return meanPoisson(obs)
}

func (p *Poisson) Variance(obs []float64, mean float64) float64 {
	return variancePoisson(obs, mean)
}

func (p *Poisson) NewSeries() (metric.SeriesRecorder, error) {
	series, done, err := metric.NewSampledSeries(p.capacity, p.window, p.strategy)
	if err != nil {
		return nil, fmt.Errorf("failed to create Poisson PDF estimator: %v", err)
	}
	p.done = done
	return series, nil
}

func (p *Poisson) Transform(obs float64) float64 {
	return obs
}

func (p *Poisson) K() (float64, error) {
	return p.k.CalculateP()
}

func (p *Poisson) Done() {
	p.done()
}

func (p *Poisson) String() string {
	return "poisson"
}

// NewPoisson returns a new Poisson distribution which bootstraps the test using capacity number of samples and combines
// observations occuring within each sampleWindow using the strategy given, such as SampleSum, SampleAvg, SampleMax, etc.
// K can be set to maintain a particular error rate or as a fixed value.
func NewPoisson(capacity int, sampleWindow time.Duration, strategy func([]float64) float64, k K) *Poisson {
	return &Poisson{
		capacity: capacity,
		window:   sampleWindow,
		strategy: strategy,
		k:        k,
	}
}

// LogNormal returns a new log normal distribution for metrics where performance has a long-tail nature, such as latency.
type LogNormal struct {
	capacity int
	k        K
}

func (p *LogNormal) Mean(obs []float64) float64 {
	return meanNormal(obs)
}

func (p *LogNormal) Variance(obs []float64, mean float64) float64 {
	return varianceNormal(obs, mean)
}

func (p *LogNormal) NewSeries() (metric.SeriesRecorder, error) {
	return metric.NewSeries(p.capacity)
}

func (p *LogNormal) Transform(obs float64) float64 {
	return math.Log(obs)
}

func (p *LogNormal) K() (float64, error) {
	return p.k.CalculateLN()
}

func (p *LogNormal) Done() {}

func (p *LogNormal) String() string {
	return "log-normal"
}

// NewLogNormal returns a log normal estimator bootstrapped from capacity initial observations where K is set to approximate
// a desired Type I error rate
func NewLogNormal(capacity int, k K) *LogNormal {
	return &LogNormal{
		capacity: capacity,
		k:        k,
	}
}

func meanNormal(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	s := 0.0
	for _, v := range values {
		s = s + v
	}
	return s / float64(len(values))
}

func varianceNormal(values []float64, mean float64) float64 {
	s := 0.0
	for _, v := range values {
		s = s + math.Pow(v-mean, 2)
	}
	return s / float64(len(values)-1)
}

func meanPoisson(values []float64) float64 {
	// MLE for Poisson mean is the same as Normal, mean of the observed sample
	return meanNormal(values)
}

func variancePoisson(values []float64, mean float64) float64 {
	// Poisson variance also equal to lambda, so return the MLE of lambda already calculated from
	// the observed values
	return mean
}
