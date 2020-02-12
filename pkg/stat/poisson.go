package stat

import (
	"fmt"
	"time"

	"github.com/BTBurke/monny/pkg/fsm"
	"github.com/BTBurke/monny/pkg/metric"
)

var _ Test = &PoissonTest{}

// PoissonTest applies one or more test statistics to a series of observations that is assumed to be poisson
// distributed, such as error rates over a sampling window. It initially looks for changes from background in the positive direction (increasing error rates, etc.)
// Once in an alarm condition, you must manually transition it to a new state to start testing for changes in the other direction (e.g., self correcting
// temporary changes in error rates, etc.)
type PoissonTest struct {
	name metric.Name
	sub  []Statistic
}

// PoissonOption applies options to construct a custom estimator
type PoissonOption func(*PoissonTest) error

func (t *PoissonTest) Name() string {
	return t.name.String()
}

func (t *PoissonTest) Record(obs float64) error {
	for _, s := range t.sub {
		if err := s.Record(obs); err != nil {
			return err
		}
	}
	return nil
}

func (t *PoissonTest) Transition(state fsm.State, reset bool) error {
	for _, s := range t.sub {
		if err := s.Transition(state, reset); err != nil {
			return err
		}
	}
	return nil
}

func (t *PoissonTest) HasAlarmed() bool {
	for _, s := range t.sub {
		if s.HasAlarmed() {
			return true
		}
	}
	return false
}

func (t *PoissonTest) State() []fsm.State {
	out := make([]fsm.State, 0)
	for _, s := range t.sub {
		out = append(out, s.State())
	}
	return out
}

func (t *PoissonTest) Done() {
	for _, s := range t.sub {
		s.Done()
	}
}

// NewPoissonTest will return a new test statistic for log poisson distributed values.  If no options are applied, it will default to testing
// using both Shewart and standard EWMA tests in parallel.
func NewPoissonTest(name metric.Name, opts ...PoissonOption) (*PoissonTest, error) {
	e := &PoissonTest{name: name}
	for _, opt := range opts {
		if err := opt(e); err != nil {
			return nil, fmt.Errorf("failed to apply option to PoissonTest: %v", err)
		}
	}
	if len(e.sub) == 0 {
		e.sub = append(e.sub, DefaultPoissonEWMA(), DefaultPoissonShewart())
	}
	return e, nil
}

// WithPoissonStatistic will use a custom estimator.  If this is used, no default estimators will be used.
func WithPoissonStatistic(e *TestStatistic) PoissonOption {
	return func(l *PoissonTest) error {
		l.sub = append(l.sub, e)
		return nil
	}
}

// DefaultPoissonEWMA constructs a default EWMA estimator with window 50 observations, lambda 0.25, k 3.0, poisson distribution with
// a sample window of 15 seconds
func DefaultPoissonEWMA() *TestStatistic {
	est, _ := NewEWMATestStatistic("ewma", .25, &FixedK{5.5}, NewPoisson(50, 15*time.Second, metric.SampleSum))
	return est
}

// DefaultPoissonShewart constructs a default EWMA estimator for Shewart with window 50 observations, lambda 1.0, k 3.0, poisson distribution with
// a sample window of 15 seconds
func DefaultPoissonShewart() *TestStatistic {
	est, _ := NewEWMATestStatistic("shewart", 1.0, &FixedK{5.5}, NewPoisson(50, 15*time.Second, metric.SampleSum))
	return est
}

// Metric will return current values from all sub estimators.  It defines the following metrics identified by metadata:
// <log field>[strategy=<(ewma|shewart)> type=estimator value=<(current|limit>]
//
// This gives the current value of the estimator and the testing limit.  This can be plotted as a spark line with the current
// testing limit.
//
// Example: request_error_rate[loc=us-west-1 host=host1 pdf=log-normal type=estimator strategy=ewma value=current] 3.455654543
//          request_error_rate[loc=us-west-1 host=host1 pdf=log-normal type=estimator strategy=ewma value=limit] 4.2435454343
func (e *PoissonTest) Metric() map[string]float64 {
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
