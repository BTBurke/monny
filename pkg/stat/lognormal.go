package stat

import (
	"fmt"

	"github.com/BTBurke/monny/pkg/fsm"
	"github.com/BTBurke/monny/pkg/metric"
)

var _ Test = &LogNormalTest{}

// LogNormalTest applies one or more test statistics to a series of observations that is assumed to be log-normally
// distributed.  It initially looks for changes from background in the positive direction (increasing latencies, etc.)
// Once in an alarm condition, you must manually transition it to a new state to start testing for changes in the other direction (e.g., self correcting
// temporary changes in latencies, etc.)
type LogNormalTest struct {
	name metric.Name
	sub  []Statistic
}

// LogNormalOption applies options to construct a custom estimator
type LogNormalOption func(*LogNormalTest) error

func (t *LogNormalTest) Name() string {
	return t.name.String()
}

func (t *LogNormalTest) Record(obs float64) error {
	for _, s := range t.sub {
		if err := s.Record(obs); err != nil {
			return err
		}
	}
	return nil
}

func (t *LogNormalTest) Transition(state fsm.State, reset bool) error {
	for _, s := range t.sub {
		if err := s.Transition(state, reset); err != nil {
			return err
		}
	}
	return nil
}

func (t *LogNormalTest) HasAlarmed() bool {
	for _, s := range t.sub {
		if s.HasAlarmed() {
			return true
		}
	}
	return false
}

func (t *LogNormalTest) State() []fsm.State {
	out := make([]fsm.State, 0)
	for _, s := range t.sub {
		out = append(out, s.State())
	}
	return out
}

func (t *LogNormalTest) Done() {
	for _, s := range t.sub {
		s.Done()
	}
}

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
	return func(l *LogNormalTest) error {
		l.sub = append(l.sub, e)
		return nil
	}
}

// DefaultLogNormalEWMA constructs a default EWMA estimator with window 50 observations, lambda 0.25, k 3.0, log normal distribution
func DefaultLogNormalEWMA() *TestStatistic {
	est, _ := NewEWMATestStatistic("ewma", .25, 0.05, NewLogNormal(50))
	return est
}

// DefaultLogNormalShewart constructs a default EWMA estimator for Shewart with window 50 observations, lambda 1.0, k 3.0, log normal distribution
func DefaultLogNormalShewart() *TestStatistic {
	est, _ := NewEWMATestStatistic("shewart", 1.0, 0.05, NewLogNormal(50))
	return est
}

// Metric will return current values from all sub estimators.  It defines the following metrics identified by metadata:
// <log field>[strategy=<(ewma|shewart)> type=estimator value=<(current|limit>]
//
// This gives the current value of the estimator and the testing limit.  This can be plotted as a spark line with the current
// testing limit.
//
// Example: disk_latency[loc=us-west-1 host=host1 pdf=log-normal type=estimator strategy=ewma value=current] 3.455654543
//          disk_latency[loc=us-west-1 host=host1 pdf=log-normal type=estimator strategy=ewma value=limit] 4.2435454343
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
