package fsm

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNonce(t *testing.T) {

	n := make([]byte, 16)
	_, err := rand.Read(n)
	assert.NoError(t, err)

	var tt = []struct {
		nonce       []byte
		receive     []byte
		shouldError bool
	}{
		{nonce: n, receive: n, shouldError: false},
		{nonce: n, receive: []byte{}, shouldError: true},
	}
	for _, t1 := range tt {
		non := &nonce{
			current:  t1.nonce,
			received: t1.receive,
		}
		err := non.ok()
		switch t1.shouldError {
		case true:
			assert.Error(t, err)
		default:
			assert.NoError(t, err)
		}
		// second checks with same nonce should error every time
		err2 := non.ok()
		assert.Error(t, err2)
	}

}

func TestMachineNonce(t *testing.T) {
	m, err := NewMachineNonce(State("initial"), WithTransitions(
		T(State("initial"), State("processing")),
		T(State("processing"), State("error"), State("finished")),
	))
	assert.NoError(t, err)
	assert.Equal(t, m.m.current, State("initial"))
	assert.Equal(t, m.m.initial, State("initial"))
	assert.True(t, m.Allowable(m.State(), State("processing")))
	assert.False(t, m.Allowable(m.State(), State("finished")))
	assert.NoError(t, m.Transition(State("processing"), m.nonce.current))
	assert.Error(t, m.Transition(State("initial"), m.nonce.current))
	assert.Equal(t, m.m.current, State("processing"))
	assert.NoError(t, m.Transition("finished", m.nonce.current))
}

func TestMachineBadNonce(t *testing.T) {
	m, err := NewMachineNonce(State("initial"), WithTransitions(
		T(State("initial"), State("processing")),
		T(State("processing"), State("error"), State("finished")),
	))
	assert.NoError(t, err)
	assert.Equal(t, m.m.current, State("initial"))
	assert.Equal(t, m.m.initial, State("initial"))
	assert.True(t, m.Allowable(m.State(), State("processing")))
	assert.False(t, m.Allowable(m.State(), State("finished")))
	// bad nonce stops allowable transition and generates a new nonce
	n1 := m.nonce.current
	assert.Error(t, m.Transition(State("processing"), []byte{}))
	assert.Equal(t, m.m.current, State("initial"))
	// without WithStoppable, FSM can transition with correct nonce, which should be different
	// than the previous try
	n2 := m.nonce.current
	assert.NotEqual(t, n1, n2)
	assert.NoError(t, m.Transition("processing", m.nonce.current))
}

func TestMachineBadNonceStoppable(t *testing.T) {
	m, err := NewMachineNonce(State("initial"), WithStoppable(), WithTransitions(
		T(State("initial"), State("processing")),
		T(State("processing"), State("error"), State("finished")),
	))
	assert.NoError(t, err)
	assert.Equal(t, m.m.current, State("initial"))
	assert.Equal(t, m.m.initial, State("initial"))
	assert.True(t, m.Allowable(m.State(), State("processing")))
	assert.False(t, m.Allowable(m.State(), State("finished")))
	// bad nonce stops allowable transition and generates a new nonce
	n1 := m.nonce.current
	assert.Error(t, m.Transition(State("processing"), []byte{}))
	assert.Equal(t, m.m.current, State("initial"))
	// with WithStoppable, FSM cannot transition even with correct nonce, which should be different
	// than the previous try
	n2 := m.nonce.current
	assert.NotEqual(t, n1, n2)
	assert.Error(t, m.Transition(State("processing"), m.nonce.current))
	// should be able to reset, new nonce should be generated and transition is allowed
	err = m.Reset()
	assert.NoError(t, err)
	n3 := m.nonce.current
	assert.NotEqual(t, n2, n3)
	assert.NoError(t, m.Transition(State("processing"), m.nonce.current))
}
