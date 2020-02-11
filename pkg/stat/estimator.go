package stat

import (
	"fmt"
	"math"

	"github.com/BTBurke/monny/pkg/fsm"
	"github.com/BTBurke/monny/pkg/metric"
)

type TestStatistic struct {
	name    string
	lambda  float64
	k       *K
	limit   float64
	series  metric.SeriesRecorder
	fsm     *fsm.Machine
	current float64
	pdf     PDF
}

func (e *TestStatistic) Name() string {
	return e.name
}

func (e *TestStatistic) Value() float64 {
	return e.current
}

func (e *TestStatistic) Limit() float64 {
	return e.limit
}

func (e *TestStatistic) Done() {
	e.pdf.Done()
}

func (e *TestStatistic) Record(o float64) error {
	o = e.pdf.Transform(o)
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
			e.series.Reset()
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
		if e.series.Count() >= e.series.Capacity() {
			values := e.series.Values()
			mean := e.pdf.Mean(values)
			variance := e.pdf.Variance(values, mean)
			if mean > 0.0 && variance > 0.0 {
				if err := e.fsm.Transition(TestingUCL); err != nil {
					return err
				}
				e.current = mean
				e.limit = calculateLimit(mean, variance, e.lambda, e.k, 1)
			}
		}
	case LCLInitial:
		if e.series.Count() >= e.series.Capacity() {
			values := e.series.Values()
			mean := e.pdf.Mean(values)
			variance := e.pdf.Variance(values, mean)
			if mean > 0.0 && variance > 0.0 {
				if err := e.fsm.Transition(TestingLCL); err != nil {
					return err
				}
				e.current = mean
				e.limit = calculateLimit(mean, variance, e.lambda, e.k, -1)
			}
		}
	}
	return nil
}

// HasAlarmed returns true if the estimator has detected that the current value of the statistic has exceeded either
// the UCL or LCL.  This will continue to return true until the estimator is manually transitioned to a new state.
func (e *TestStatistic) HasAlarmed() bool {
	switch e.State() {
	case UCLTrip, LCLTrip:
		return true
	default:
		return false
	}
}

// State returns the current state of the estimator
func (e *TestStatistic) State() fsm.State {
	return e.fsm.State()
}

// caluculate the current value of the test statistic
func (e *TestStatistic) calculateCurrent(o float64) {
	e.current = (e.lambda * o) + ((1.0 - e.lambda) * e.current)
}

// Transition will attempt to transition to estimator to the desired state.  Optionally reset the series to
// force it to collect new baseline observations before entering testing phase
func (e *TestStatistic) Transition(state fsm.State, resetSeries bool) error {
	if resetSeries {
		e.series.Reset()
	}
	return e.fsm.Transition(state)
}

// calculateLimit will determine the UCL or LCL limit (UCL => direction +1, LCL => direction -1)
// sensitivity is a float within +/- 1.0 that adjusts limits to create a more senstive alarm if sensitivity > 0.0 or less
// sensitive if < 0.0
func calculateLimit(mean float64, variance float64, lambda float64, k *K, direction int) float64 {
	estimatorVariance := (lambda / (2.0 - lambda)) * variance

	kc, err := k.Calculate()
	if err != nil {
		kc = 5.7
	}

	switch {
	// +1 calculate UCL, -1 LCL
	case direction >= 0:
		return mean + (kc * math.Sqrt(estimatorVariance))
	default:
		return mean - (kc * math.Sqrt(estimatorVariance))
	}
}

// NewEWMATestStatistic returns a new EWMA test statistic.  Transform can be used to apply a function to each raw observation before
// it is tested by the statistic.  e.g., for log-normally distributed observations, the transform would be math.Log(observation)
func NewEWMATestStatistic(name string, lambda float64, type1Error float64, pdf PDF) (*TestStatistic, error) {
	series, err := pdf.NewSeries()
	if err != nil {
		return nil, fmt.Errorf("unable to create EWMA test statistic for %s: %v", pdf.String(), err)
	}
	machine, err := newMachine(UCLInitial)
	if err != nil {
		return nil, fmt.Errorf("failed to create estimator FSM: %v", err)
	}
	return &TestStatistic{
		name:   name,
		k:      &K{type1Error},
		lambda: lambda,
		series: series,
		fsm:    machine,
		pdf:    pdf,
	}, nil
}
