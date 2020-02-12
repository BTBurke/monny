package stat

import (
	"fmt"
	"math"
)

//go:generate go run calibrate.go

type K interface {
	Calculate() (float64, error)
}

const (
	// X1,X2 are magic numbers that define a least squares regression on the performance of the EWMA
	// based on desired Type I false alarm rate
	X1 float64 = 17.0165
	X2 float64 = -3.7986
)

// K calculates the appropriate k value for the EWMA limit equation based on a desired type I error rate
type KErrorRate struct {
	// e is the desired type I error probability
	e float64
}

// Calculate returns the value of k given the desired error rate
func (k *KErrorRate) Calculate() (float64, error) {
	kestimate := (math.Log(k.e) - X1) / X2
	if math.IsNaN(kestimate) || math.IsInf(kestimate, 1) || math.IsInf(kestimate, -1) {
		return 0, fmt.Errorf("can not calculate k for error value: %f", k.e)
	}
	return kestimate, nil
}

func (l *KErrorRate) SetError(e float64) {
	l.e = e
}

type FixedK struct {
	k float64
}

func (k *FixedK) Calculate() (float64, error) {
	return k.k, nil
}

func NewFixedK(k float64) *FixedK {
	return &FixedK{k}
}
