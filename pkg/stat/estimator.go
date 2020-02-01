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
	// transform will apply an initial transformation to the observed value before calculating the statistic
	// e.g., apply ln(observation) if values are expected to be log-normally distributed
	transform func(float64) float64
}

func (e *Estimator) SetTransform(transform func(float64) float64) {
	e.transform = transform
}

func (e *Estimator) SetSensitivty(sensitivity float64) {
	switch {
	case sensitivity > 1.0:
		sensitivity = 1.0
	case sensitivity < -1.0:
		sensitivity = -1.0
	default:
	}
	e.sensitivity = sensitivity
}

func (e *Estimator) Name() string {
	return e.name
}

func (e *Estimator) Value() float64 {
	return e.current
}

func (e *Estimator) Limit() float64 {
	return e.limit
}

func (e *Estimator) Record(o float64) error {
	if e.transform != nil {
		o = e.transform(o)
	}
	if math.IsNaN(o) || math.IsInf(o, 1) || math.IsInf(o, -1) {
		return fmt.Errorf("transform(value) is not defined")
	}

	e.series.Record(o)
	switch e.fsm.State() {
	case Reset:
		// reset assumes that estimator should be restarted from steady state non-alarmed condition
		// and next transition should be to testing the UCL
		if err := e.fsm.Transition(UCLInitial); err != nil {
			return err
		}
		// if forced into reset with existing observations, start series recording over again
		if e.series.Count() > 0 {
			series, err := metric.NewSeries(e.window)
			if err != nil {
				return fmt.Errorf("failed to create new series for estimator: %v", err)
			}
			e.series = series
			// re-record the current observation after reseting the series
			e.series.Record(o)
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

// HasAlarmed returns true if the estimator has detected that the current value of the statistic has exceeded either
// the UCL or LCL.  This will continue to return true until the estimator is manually transitioned to a new state.
func (e *Estimator) HasAlarmed() bool {
	switch e.State() {
	case UCLTrip, LCLTrip:
		return true
	default:
		return false
	}
}

// State returns the current state of the estimator
func (e *Estimator) State() fsm.State {
	return e.fsm.State()
}

// caluculate the current value of the test statistic
func (e *Estimator) calculateCurrent(o float64) {
	e.current = (e.lambda * o) + ((1.0 - e.lambda) * e.current)
}

// Transition will attempt to transition to estimator to the desired state.  Optionally reset the series to
// force it to collect new baseline observations before entering testing phase
func (e *Estimator) Transition(state fsm.State, resetSeries bool) error {
	if resetSeries {
		series, err := metric.NewSeries(e.window)
		if err != nil {
			return fmt.Errorf("failed to reset series on transition: %v", err)
		}
		e.series = series
	}
	return e.fsm.Transition(state)
}

// calculateLimit will determine the UCL or LCL limit (UCL => direction +1, LCL => direction -1)
// sensitivity is a float within +/- 1.0 that adjusts limits to create a more senstive alarm if sensitivity > 0.0 or less
// sensitive if < 0.0
func calculateLimit(mean float64, variance float64, lambda float64, k float64, sensitivity float64, direction int) float64 {
	estimatorVariance := (lambda / (2.0 - lambda)) * variance

	// adjust sensitivity from base 0.0, limits of +/- 1.0, >0.0 moves the limit closer to the initial mean of the statistic
	// <0.0 creates a higher limit
	switch {
	case sensitivity > 1.0:
		sensitivity = 1.0
	case sensitivity < -1.0:
		sensitivity = -1.0
	default:
	}
	k = k * (1.0 - sensitivity)

	switch {
	// +1 calculate UCL, -1 LCL
	case direction >= 0:
		return mean + (k * math.Sqrt(estimatorVariance))
	default:
		return mean - (k * math.Sqrt(estimatorVariance))
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

// NewEWMAEstimator returns a new EWMA estimator.  Transform can be used to apply a function to each raw observation before
// it is tested by the test statistic.  e.g., for log-normally distributed observations, the transform would be math.Log(observation)
func NewEWMAEstimator(name string, window int, lambda float64, k float64, transform func(float64) float64) (*Estimator, error) {
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
		sensitivity: 0.0,
		transform:   transform,
	}, nil
}
