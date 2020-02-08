package metric

import (
	"fmt"
	"math"
	"sync"
	"time"
)

var _ SeriesRecorder = &SampledSeries{}

type SampledSeries struct {
	s         *Series
	mu        sync.RWMutex
	t         *time.Ticker
	obs       []float64
	transform func([]float64) float64
	done      chan bool
	wg        sync.WaitGroup
}

func NewSampledSeries(capacity int, sampleWindow time.Duration, transform func([]float64) float64, opts ...SeriesOption) (*SampledSeries, func(), error) {
	s, err := NewSeries(capacity, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("could not create sampled series: %v", err)
	}
	if sampleWindow == 0 {
		return nil, nil, fmt.Errorf("sampled series window must be greater than zero")
	}

	ss := &SampledSeries{
		s:         s,
		t:         time.NewTicker(sampleWindow),
		obs:       make([]float64, 0),
		transform: transform,
		done:      make(chan bool),
	}
	ss.wg.Add(1)
	go func(s *SampledSeries) {
		defer s.wg.Done()
		for {
			select {
			case <-s.t.C:
				s.mu.Lock()
				if len(s.obs) == 0 {
					s.s.Record(0.0)
				} else {
					s.s.Record(s.transform(s.obs))
					s.obs = make([]float64, 0)
				}
				s.mu.Unlock()
			case <-s.done:
				s.t.Stop()
				return
			}
		}
	}(ss)

	return ss, func() { ss.done <- true; ss.wg.Wait() }, nil
}

func (s *SampledSeries) Record(obs float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.obs = append(s.obs, obs)
}

func (s *SampledSeries) Values() []float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.s.Values()
}

func (s *SampledSeries) Name() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.s.Name()
}

func (s *SampledSeries) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.s.Count()
}

func SampleAverage(obs []float64) float64 {
	if len(obs) == 0 {
		return 0.0
	}
	return SampleSum(obs) / float64(len(obs))
}

func SampleMin(obs []float64) float64 {
	if len(obs) == 0 {
		return 0.0
	}
	min := obs[0]
	for _, o := range obs {
		min = math.Min(min, o)
	}
	return min
}

func SampleMax(obs []float64) float64 {
	if len(obs) == 0 {
		return 0.0
	}
	max := obs[0]
	for _, o := range obs {
		max = math.Max(max, o)
	}
	return max
}

func SampleSum(obs []float64) float64 {
	sum := 0.0
	for _, o := range obs {
		sum += o
	}
	return sum
}
