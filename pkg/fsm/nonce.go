package fsm

import (
	"bytes"
	"crypto/rand"
	"sync"
)

// MachineNonce represents a FSM with transitions that are protected by a nonce
type MachineNonce struct {
	m     *Machine
	nonce *nonce
}

// NewMachineNonce returns a new FSM with transitions protected by a nonce that must be
// supplied on each state transition.  Use the same MachineOptions as Machine to configure
// allowable transitions and stop criteria.
func NewMachineNonce(initial State, opts ...MachineOption) (*MachineNonce, error) {
	m, err := NewMachine(initial, opts...)
	if err != nil {
		return nil, err
	}
	n, err := newNonce()
	if err != nil {
		return nil, err
	}
	return &MachineNonce{
		m:     m,
		nonce: n,
	}, nil
}

// State returns the current state of the machine
func (m *MachineNonce) State() State {
	return m.m.State()
}

// Allowable checks whether a transition between two states is allowed
func (m *MachineNonce) Allowable(from, to State) bool {
	return m.m.Allowable(from, to)
}

func (m *MachineNonce) reset() error {
	m.m.reset()
	n, err := newNonce()
	if err != nil {
		return err
	}
	m.nonce = n
	return nil
}

// Reset will return the machine to its initial state and clear any stop condition if it exists
func (m *MachineNonce) Reset() error {
	return m.reset()
}

// Transition will change the state of the FSM if the supplied nonce matches the current
// value
func (m *MachineNonce) Transition(to State, nonce []byte) error {
	m.nonce.received = nonce
	if err := m.m.transition(to, m.m.stoppable, m.nonce); err != nil {
		n, _ := newNonce()
		m.nonce = n
		return err
	}
	n, err := newNonce()
	if err != nil {
		return err
	}
	m.nonce = n
	return nil
}

// Nonce returns the current nonce value
func (m *MachineNonce) Nonce() []byte {
	return m.nonce.current
}

type nonce struct {
	current  []byte
	once     sync.Once
	received []byte
}

func (n *nonce) ok() error {
	res := struct{ match bool }{}
	f := makeNonceCheck(n.current, n.received, &res)
	n.once.Do(f)
	switch res.match {
	case true:
		return nil
	default:
		return NonceError{Msg: "nonce did not match"}
	}
}

func newNonce() (*nonce, error) {
	n := make([]byte, 16)
	_, err := rand.Read(n)
	if err != nil {
		return nil, err
	}
	return &nonce{
		current: n,
	}, nil
}

func makeNonceCheck(current []byte, received []byte, res *struct{ match bool }) func() {
	return func() {
		res.match = bytes.Equal(current, received)
	}
}
