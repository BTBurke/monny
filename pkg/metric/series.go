package metric

import (
	"fmt"
	"math"
)

type Series struct {
	name   Name
	count  int
	values []float64
}

type SeriesOption func(s *Series) error

// Values returns a copy of the current values in the series in temporal order from oldest to most recent
func (s *Series) Values() []float64 {
	switch {
	case s.count < len(s.values):
		out := make([]float64, len(s.values))
		copy(out, s.values)
		return out
	default:
		out := make([]float64, 0, len(s.values))
		oldest := s.nextIndex()
		return append(append(out, s.values[oldest:]...), s.values[0:oldest]...)
	}
}

// Record adds a new observation to the series
func (s *Series) Record(p float64) {
	if len(s.values) == 0 {
		return
	}

	s.values[s.nextIndex()] = p
	s.count++
}

// nextIndex returns the index of the oldest observation in the series to be overwritten by new data
func (s *Series) nextIndex() int {
	cap := len(s.values)
	if cap == 0 {
		return 0
	}
	return int(math.Mod(float64(s.count), float64(cap)))
}

// Count returns the total number of observations for this series
func (s *Series) Count() int {
	return s.count
}

// Name returns the name of the series and associated metadata
func (s *Series) Name() string {
	return s.name.String()
}

// NewSeries creates a new series with a capacity of cap
func NewSeries(cap int, opts ...SeriesOption) (*Series, error) {
	if cap <= 0 {
		return nil, fmt.Errorf("series must be initialized with a capacity >= 1")
	}

	s := &Series{
		values: make([]float64, cap),
	}
	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// WithName sets the name of the series
func WithName(name string, md map[string]string) SeriesOption {
	return func(s *Series) error {
		if name == "" {
			return fmt.Errorf("series name must be the non-empty string")
		}
		s.name = NewName(name, md)
		return nil
	}
}

// WithValues initializes a series from an existing set of observations.  The number of observations does not
// have to be equal to the capacity.
func WithValues(values []float64) SeriesOption {
	return func(s *Series) error {
		for _, v := range values {
			s.Record(v)
		}
		return nil
	}
}
