// Package fsm implements a finite state machine used in custom streaming protocols
package fsm

import (
	"fmt"
)

// State represents a possible transition state for the FSM
type State string

// FSM is an interface that defines the operation of different types of
// state machines.  Note that the interface does not include a Transition() function
// that should be implemented separately by each concrete type of FSM.
type FSM interface {
	State() State
	Allowable(from, to State) bool

	transition(to State, guards ...transitionGuard) error
	reset() error
}

// Machine is a basic finite state machine
type Machine struct {
	current   State
	initial   State
	allowable map[State][]State
	stoppable stoppable
}

// NewMachine returns a new basic Machine with configured options.  If you do not utilize any
// options, the machine will not have any configured transitions.
func NewMachine(initial State, opts ...MachineOption) (*Machine, error) {
	machine := &Machine{
		current:   initial,
		initial:   initial,
		allowable: map[State][]State{},
	}
	for _, opt := range opts {
		if err := opt(machine); err != nil {
			return nil, err
		}
	}
	return machine, nil
}

// State returns the current state of the Machine
func (m *Machine) State() State {
	return m.current
}

// Allowable checks whether a transition between two states is allowable
func (m *Machine) Allowable(from, to State) bool {
	return contains(to, m.allowable[from])
}

// Transition will change the current state of the machine if it is allowable
func (m *Machine) Transition(to State) error {
	return m.transition(to, m.stoppable)
}

// Reset will reset the machine to its initial state and remove any stop condition if it
// exists
func (m *Machine) Reset() {
	m.reset()
}

func (m *Machine) reset() error {
	m.current = m.initial
	m.stoppable.stopped = false
	return nil
}

func (m *Machine) transition(to State, guards ...transitionGuard) error {
	for _, guard := range guards {
		if err := guard.ok(); err != nil {
			m.stoppable.stopped = true
			return err
		}
	}

	switch m.Allowable(m.current, to) {
	case true:
		m.current = to
		return nil
	default:
		m.stoppable.stopped = true
		return TransitionNotAllowed{Msg: fmt.Sprintf("cannot transition from state %s to %s", m.current, to)}
	}

}

func contains(s State, all []State) bool {
	for _, a := range all {
		if s == a {
			return true
		}
	}
	return false
}
