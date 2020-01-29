package fsm

// Transition represents an allowable transition from one state to another
type Transition struct {
	From State
	To   State
}

// transitionGuard represents a closure over a function that will stop transition on
// a particular condition beyond just whether the transition is allowed, such as a nonce
// match, etc.
type transitionGuard interface {
	ok() error
}

// T is a shorthand function for declaring allowable transitions during FSM creation
func T(from State, tos ...State) []Transition {
	var transitions []Transition
	for _, to := range tos {
		transitions = append(transitions, Transition{
			From: from,
			To:   to,
		})
	}
	return transitions
}

func flatten(t [][]Transition) []Transition {
	var transitions []Transition
	for _, t1 := range t {
		transitions = append(transitions, t1...)
	}
	return transitions
}
