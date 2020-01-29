package stat

import (
	"fmt"
	"math"

	"github.com/BTBurke/monny/pkg/fsm"
	"github.com/BTBurke/monny/pkg/metric"
)

type Estimator struct {
	name        string
	window      int
	lambda      float64
	k           float64
	limit       float64
	series      *metric.Series
	fsm         *fsm.Machine
	current     float64
	EWMA0       float64
	variance0   float64
	sensitivity float64
}

func (e *Estimator) Record(o float64) error {
	o = math.Log(o)
	if math.IsNaN(o) || math.IsInf(o, 1) || math.IsInf(o, -1) {
		return fmt.Errorf("ln(value) is not defined")
	}

	e.series.Record(o)
	switch e.fsm.State() {
	case Reset:
		if err := e.fsm.Transition(UCLInitial); err != nil {
			return err
		}
		fallthrough
	case TestingUCL:
		e.calculateCurrent(o)
		if e.current >= e.limit {
			if err := e.fsm.Transition(UCLTrip); err != nil {
				return err
			}
		}
	case TestingLCL:
		e.calculateCurrent(o)
		if e.current <= e.limit {
			if err := e.fsm.Transition(LCLTrip); err != nil {
				return err
			}
		}
	case UCLInitial:
		if e.series.Count() >= e.window {
			values := e.series.Values()
			e.EWMA0 = mean(values)
			e.variance0 = variance(values, e.EWMA0)
			if e.EWMA0 > 0.0 && e.variance0 > 0.0 {
				if err := e.fsm.Transition(TestingUCL); err != nil {
					return err
				}
				e.current = e.EWMA0
				e.limit = calculateLimit(e.EWMA0, e.variance0, e.lambda, e.k, e.sensitivity, 1)
			}
		}
	case LCLInitial:
		if e.series.Count() >= e.window {
			values := e.series.Values()
			e.EWMA0 = mean(values)
			e.variance0 = variance(values, e.EWMA0)
			if e.EWMA0 > 0.0 && e.variance0 > 0.0 {
				if err := e.fsm.Transition(TestingLCL); err != nil {
					return err
				}
				e.current = e.EWMA0
				e.limit = calculateLimit(e.EWMA0, e.variance0, e.lambda, e.k, e.sensitivity, -1)
			}
		}
	}
	return nil
}

func (e *Estimator) HasAlarmed() bool {
	switch e.State() {
	case UCLTrip, LCLTrip:
		return true
	default:
		return false
	}
}

func (e *Estimator) State() fsm.State {
	return e.fsm.State()
}

func (e *Estimator) calculateCurrent(o float64) {
	e.current = (e.lambda * o) + ((1.0 - e.lambda) * e.current)
}

func (e *Estimator) Transition(state fsm.State) error {
	return e.fsm.Transition(state)
}

func calculateLimit(mean float64, variance float64, lambda float64, k float64, sensitivity float64, direction int) float64 {
	estimatorVariance := (lambda / (2.0 - lambda)) * variance
	// +1 calculate UCL, -1 LCL
	switch {
	case direction >= 0:
		return sensitivity * (mean + (k * math.Sqrt(estimatorVariance)))
	default:
		return (1.0 / sensitivity) * (mean - (k * math.Sqrt(estimatorVariance)))
	}
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	s := 0.0
	for _, v := range values {
		s = s + v
	}
	return s / float64(len(values))
}

func variance(values []float64, mean float64) float64 {
	s := 0.0
	for _, v := range values {
		s = s + math.Pow(v-mean, 2)
	}
	return s / float64(len(values)-1)
}

func NewEstimator(name string, window int, lambda float64, k float64) (*Estimator, error) {
	series, err := metric.NewSeries(window)
	if err != nil {
		return nil, fmt.Errorf("failed to create estimator: %v", err)
	}
	machine, err := newMachine(UCLInitial)
	if err != nil {
		return nil, fmt.Errorf("failed to create estimator FSM: %v", err)
	}
	return &Estimator{
		name:        name,
		window:      window,
		k:           k,
		lambda:      lambda,
		series:      series,
		fsm:         machine,
		sensitivity: 1.0,
	}, nil
}

func defaultEWMA() *Estimator {
	est, _ := NewEstimator("ewma", 50, .25, 3.0)
	return est
}

func defaultShewart() *Estimator {
	est, _ := NewEstimator("shewart", 50, 1.0, 3.0)
	return est
}
