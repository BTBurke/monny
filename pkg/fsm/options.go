package fsm

// MachineOption represents options to initially set up a machine
type MachineOption func(m *Machine) error

// WithTransition allows the addition of a single edge on the transition graph.  To add multiple
// edges at once, try WithTransitions.
func WithTransition(t Transition) MachineOption {
	return func(m *Machine) error {
		m.allowable[t.From] = append(m.allowable[t.From], t.To)
		return nil
	}
}

// WithTransitions will allow the addition of multiple transitions using the T(from, to...) short
// function.  For example, you can call `NewMachine(Initial, WithTransitions(T(One, Two, Three), T(Two, Three)))`
func WithTransitions(transitions ...[]Transition) MachineOption {
	return func(m *Machine) error {
		trans := flatten(transitions)
		for _, t := range trans {
			m.allowable[t.From] = append(m.allowable[t.From], t.To)
		}
		return nil
	}
}

// WithStoppable makes the state machine stop after an unallowable transition.  Further attempted transitions
// will always error.  You can use `Reset()` to reset the FSM to the initial state and clear the stop condition.
func WithStoppable() MachineOption {
	return func(m *Machine) error {
		m.stoppable.stopOnError = true
		return nil
	}
}

type stoppable struct {
	stopOnError bool
	stopped     bool
}

func (s stoppable) ok() error {
	if s.stopOnError && s.stopped {
		return StopError{Msg: "state machine is in stopped state due to unallowable transition"}
	}
	return nil
}
