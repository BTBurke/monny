package rng

// RNG is a random number generator
type RNG interface {
	Rand() float64
}
