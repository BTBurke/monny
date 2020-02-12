package stat

import (
	"fmt"
	"time"

	"github.com/BTBurke/monny/pkg/metric"
)

// NewPoissonTest will return a new test statistic for poisson distributed values.  If no options are applied, it will default to testing
// using both Shewart and standard EWMA tests in parallel.
func NewPoissonTest(name metric.Name, opts ...TestOption) (*Test, error) {
	e := &Test{name: name}
	for _, opt := range opts {
		if err := opt(e); err != nil {
			return nil, fmt.Errorf("failed to apply option to poisson test: %v", err)
		}
	}
	if len(e.sub) == 0 {
		e.sub = append(e.sub, DefaultPoissonEWMA(), DefaultPoissonShewart())
	}
	return e, nil
}

// DefaultPoissonEWMA constructs a default EWMA estimator with 50 bootstrap observations, lambda 0.25, error rate .05, samples added over
// 15 second windows
func DefaultPoissonEWMA() *TestStatistic {
	est, _ := NewEWMAStatistic("ewma", .25, NewPoisson(50, 15*time.Second, metric.SampleSum, KErrorRate(0.05)))
	return est
}

// DefaultPoissonShewart constructs a default shewart estimator with 50 bootstrap observations, lambda 0.25, error rate .05, samples added over
// 15 second windows
func DefaultPoissonShewart() *TestStatistic {
	est, _ := NewEWMAStatistic("shewart", 1.0, NewPoisson(50, 15*time.Second, metric.SampleSum, KErrorRate(0.05)))
	return est
}
