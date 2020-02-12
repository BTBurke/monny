package stat

import (
	"fmt"

	"github.com/BTBurke/monny/pkg/metric"
)

// NewLogNormalTest will return a new test statistic for log normally distributed values.  If no options are applied, it will default to testing
// using both Shewart and standard EWMA tests in parallel.
func NewLogNormalTest(name metric.Name, opts ...TestOption) (*Test, error) {
	e := &Test{name: name}
	for _, opt := range opts {
		if err := opt(e); err != nil {
			return nil, fmt.Errorf("failed to apply option to log normal test: %v", err)
		}
	}
	if len(e.sub) == 0 {
		e.sub = append(e.sub, DefaultLogNormalEWMA(), DefaultLogNormalShewart())
	}
	return e, nil
}

// DefaultLogNormalEWMA constructs a default EWMA estimator with window 50 observations, lambda 0.25, k 3.0, log normal distribution
func DefaultLogNormalEWMA() *TestStatistic {
	est, _ := NewEWMAStatistic("ewma", .25, NewLogNormal(50, KErrorRate(0.05)))
	return est
}

// DefaultLogNormalShewart constructs a default EWMA estimator for Shewart with window 50 observations, lambda 1.0, k 3.0, log normal distribution
func DefaultLogNormalShewart() *TestStatistic {
	est, _ := NewEWMAStatistic("shewart", 1.0, NewLogNormal(50, KErrorRate(0.05)))
	return est
}
