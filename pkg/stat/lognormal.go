package stat

import (
	"fmt"

	"github.com/BTBurke/monny/pkg/fsm"
	"github.com/BTBurke/monny/pkg/metric"
)

type EstimatorI interface {
	Record(s float64) error
	State() fsm.State
	Transition(s fsm.State) error
	HasAlarmed() bool
}

type LogNormalEstimator struct {
	name metric.Name
	sub  []EstimatorI
}

type LogNormalOption func(*LogNormalEstimator) error

func NewLogNormalEstimator(name metric.Name, opts ...LogNormalOption) (*LogNormalEstimator, error) {
	e := &LogNormalEstimator{name: name}
	for _, opt := range opts {
		if err := opt(e); err != nil {
			return nil, fmt.Errorf("failed to create LogNormalEstimator: %v", err)
		}
	}
	if len(e.sub) == 0 {
		e.sub = append(e.sub, defaultEWMA(), defaultShewart())
	}
	return e, nil
}
