package stat

import (
	"fmt"
	"math"
)

// Simulation of a stochastic process under the null hypothesis is used to experimentally determine k values for a
// desired Type I error rate for long running statistics.  These constants are recorded in kconst_gen.go.  See calibrate.go
// for the Monte Carlo simulation that calculates these constants using linear regression on observed error rates.

//go:generate go run calibrate.go

// K represents the k value in the EWMA limit calculation.  It can be set either to maintain an approximate Type I error
// rate or to a fixed value.
type K interface {
	// K value for a log normal distribution
	CalculateLN() (float64, error)
	// K value for a poisson distribution
	CalculateP() (float64, error)
}

//const (
//	X1 float64 = 17.0165
//	X2 float64 = -3.7986
//)

// KErrorRate calculates the appropriate k value for the EWMA limit equation based on a desired type I error rate
type KErrorRate float64

// Calculate returns the value of k given the desired error rate
func (k KErrorRate) CalculateLN() (float64, error) {
	return k.calculate(LogNormalA, LogNormalB)
}

func (k KErrorRate) CalculateP() (float64, error) {
	return k.calculate(PoissonA, PoissonB)
}

// calculate returns a k value based on interpolation from monte carlo simulation of expected Type I error rate
// for long running statistics.
func (k KErrorRate) calculate(a, b float64) (float64, error) {
	kest := (math.Log(float64(k)) - a) / b
	if math.IsNaN(kest) || math.IsInf(kest, 1) || math.IsInf(kest, -1) {
		return 6.5, fmt.Errorf("can not calculate k for error value: %f", k)
	}
	return float64(kest), nil
}

// KFixed is a fixed k that does not automatically adjust to maintain a particular error rate.  This is mainly useful for testing
// but can be used in cases where k is known through Monte Carlo simulation.
type KFixed float64

func (k KFixed) CalculateLN() (float64, error) {
	return float64(k), nil
}

func (k KFixed) CalculateP() (float64, error) {
	return float64(k), nil
}
