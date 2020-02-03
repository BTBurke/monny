package stat

import (
	"fmt"
	"math"

	"github.com/BTBurke/monny/pkg/fsm"
	"github.com/BTBurke/monny/pkg/metric"
)

type Statistic interface {
	Name() string
	Record(s float64) error
	State() fsm.State
	Transition(s fsm.State, reset bool) error
	HasAlarmed() bool
	Value() float64
	Limit() float64
}

// LogNormalTest applies one or more test statistics to a series of observations that is assumed to be log-normally
// distributed.  It initially looks for changes from background in the positive direction (increasing latencies, etc.)
// Once in an upper limit alarm condition, it will start testing for changes in the negative direction (e.g., self correcting
// temporary changes in latencies, etc.)
type LogNormalTest struct {
	name metric.Name
	sub  []Statistic
}

// LogNormalOption applies options to construct a custom estimator
type LogNormalOption func(*LogNormalTest) error

// NewLogNormalTest will return a new test statistic for log normally distributed values.  If no options are applied, it will default to testing
// using both Shewart and standard EWMA tests in parallel.
func NewLogNormalTest(name metric.Name, opts ...LogNormalOption) (*LogNormalTest, error) {
	e := &LogNormalTest{name: name}
	for _, opt := range opts {
		if err := opt(e); err != nil {
			return nil, fmt.Errorf("failed to apply option to LogNormalTest: %v", err)
		}
	}
	if len(e.sub) == 0 {
		e.sub = append(e.sub, DefaultLogNormalEWMA(), DefaultLogNormalShewart())
	}
	return e, nil
}

// WithLogNormalStatistic will use a custom estimator.  If this is used, no default estimators will be used.
func WithLogNormalStatistic(e *TestStatistic) LogNormalOption {
	if e.transform == nil {
		e.SetTransform(math.Log)
	}
	return func(l *LogNormalTest) error {
		l.sub = append(l.sub, e)
		return nil
	}
}

// DefaultLogNormalEWMA constructs a default EWMA estimator with window 50 observations, lambda 0.25, k 3.0, log normal distribution
func DefaultLogNormalEWMA() *TestStatistic {
	est, _ := NewEWMATestStatistic("ewma", 50, .25, 3.0, math.Log)
	return est
}

// DefaultLogNormalShewart constructs a default EWMA estimator for Shewart with window 50 observations, lambda 1.0, k 3.0, log normal distribution
func DefaultLogNormalShewart() *TestStatistic {
	est, _ := NewEWMATestStatistic("shewart", 50, 1.0, 3.0, math.Log)
	return est
}

// Metric will return current values from all sub estimators.  It defines the following metrics identified by metadata:
// <log field>[strategy=<(ewma|shewart)> type=estimator value=<(current|limit>]
//
// This gives the current value of the estimator and the testing limit.  This can be plotted as a spark line with the current
// testing limit.
//
// Example: disk_latency[loc=us-west-1 host=host1 type=estimator strategy=ewma value=current] 3.455654543
//          disk_latency[loc=us-west-1 host=host1 type=estimator strategy=ewma value=limit] 4.2435454343
func (e *LogNormalTest) Metric() map[string]float64 {
	out := make(map[string]float64)
	for _, est := range e.sub {
		nameValue := metric.NewNameFrom(e.name)
		nameValue.AddMetadata(map[string]string{"strategy": est.Name(), "type": "estimator", "value": "current"})

		nameLimit := metric.NewNameFrom(e.name)
		nameLimit.AddMetadata(map[string]string{"strategy": est.Name(), "type": "estimator", "value": "limit"})

		out[nameValue.String()] = est.Value()
		out[nameLimit.String()] = est.Limit()
	}
	return out
}
