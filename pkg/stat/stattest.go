package stat

import (
	"github.com/BTBurke/monny/pkg/fsm"
	"github.com/BTBurke/monny/pkg/metric"
)

// Test applies one or more test statistics to a series of observations that is assumed to be log-normally
// distributed.  It initially looks for changes from background in the positive direction (increasing latencies, etc.)
// Once in an alarm condition, you must manually transition it to a new state to start testing for changes in the other direction (e.g., self correcting
// temporary changes in latencies, etc.)
type Test struct {
	name metric.Name
	sub  []*TestStatistic
}

// LogNormalOption applies options to construct a custom estimator
type TestOption func(*Test) error

func (t *Test) Name() string {
	return t.name.String()
}

func (t *Test) Record(obs float64) error {
	for _, s := range t.sub {
		if err := s.Record(obs); err != nil {
			return err
		}
	}
	return nil
}

func (t *Test) Transition(state fsm.State, reset bool) error {
	for _, s := range t.sub {
		if err := s.Transition(state, reset); err != nil {
			return err
		}
	}
	return nil
}

func (t *Test) HasAlarmed() bool {
	for _, s := range t.sub {
		if s.HasAlarmed() {
			return true
		}
	}
	return false
}

func (t *Test) State() []fsm.State {
	out := make([]fsm.State, 0)
	for _, s := range t.sub {
		out = append(out, s.State())
	}
	return out
}

func (t *Test) Done() {
	for _, s := range t.sub {
		s.Done()
	}
}

// WithStatistic will use a custom estimator.  If this is used, no default estimators will be used.
func WithStatistic(e *TestStatistic) TestOption {
	return func(l *Test) error {
		l.sub = append(l.sub, e)
		return nil
	}
}

// Metric will return current values from all sub estimators.  It defines the following metrics identified by metadata:
// <log field>[strategy=<(ewma|shewart)> type=estimator value=<(current|limit>]
//
// This gives the current value of the estimator and the testing limit.  This can be plotted as a spark line with the current
// testing limit.
//
// Example: disk_latency[loc=us-west-1 host=host1 pdf=log-normal type=estimator strategy=ewma value=current] 3.455654543
//          disk_latency[loc=us-west-1 host=host1 pdf=log-normal type=estimator strategy=ewma value=limit] 4.2435454343
func (e *Test) Metric() map[string]float64 {
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
