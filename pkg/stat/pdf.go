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
	Mean(obs []float64) float64
	Variance(obs []float64, mean float64) float64
	NewSeries() (metric.SeriesRecorder, error)
	Transform(obs float64) float64
	Done()
	String() string
}

type Poisson struct {
	capacity int
	window   time.Duration
	strategy func([]float64) float64
	done     func()
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

func (p *Poisson) Done() {
	p.done()
}

func (p *Poisson) String() string {
	return "poisson"
}

func NewPoisson(capacity int, sampleWindow time.Duration, strategy func([]float64) float64) *Poisson {
	return &Poisson{
		capacity: capacity,
		window:   sampleWindow,
		strategy: strategy,
	}
}

type LogNormal struct {
	capacity int
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

func (p *LogNormal) Done() {}

func (p *LogNormal) String() string {
	return "log-normal"
}

func NewLogNormal(capacity int) *LogNormal {
	return &LogNormal{capacity}
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
