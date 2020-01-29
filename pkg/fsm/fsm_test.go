package fsm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlatten(t *testing.T) {
	t1 := Transition{
		From: State("test"),
		To:   State("success"),
	}
	t1_2 := []Transition{t1, t1}
	var tt = []struct {
		in  [][]Transition
		out []Transition
	}{
		{in: [][]Transition{t1_2, t1_2}, out: []Transition{t1, t1, t1, t1}},
	}

	for _, case1 := range tt {
		out := flatten(case1.in)
		assert.Equal(t, case1.out, out, "should flatten nested transition statements")
	}
}

func TestContains(t *testing.T) {
	var m = map[State][]State{
		State("test1"): []State{State("success"), State("failure")},
		State("test2"): []State{"failure"},
	}
	var tt = []struct {
		from   State
		to     State
		expect bool
	}{
		{from: State("test1"), to: State("success"), expect: true},
		{from: State("test1"), to: State("failure"), expect: true},
		{from: State("test1"), to: State(""), expect: false},
		{from: State("test2"), to: State("failure"), expect: true},
		{from: State("notexist"), to: State("success"), expect: false},
		{from: State(""), to: State(""), expect: false},
	}
	for _, t1 := range tt {
		out := contains(t1.to, m[t1.from])
		assert.Equal(t, out, t1.expect, "should properly find allowable transitions")
	}
}

func TestMachineCreation(t *testing.T) {
	var expect = map[State][]State{
		State("initial"):    []State{State("processing")},
		State("processing"): []State{State("error"), State("finished")},
	}
	m, err := NewMachine(State("initial"), WithTransition(Transition{State("initial"), State("processing")}),
		WithTransitions(T(State("processing"), State("error"), State("finished"))))
	assert.NoError(t, err)
	assert.Equal(t, m.allowable, expect)
}

func TestMachine(t *testing.T) {
	m, err := NewMachine(State("initial"), WithTransitions(
		T(State("initial"), State("processing")),
		T(State("processing"), State("error"), State("finished")),
	))
	assert.NoError(t, err)
	assert.Equal(t, m.current, State("initial"))
	assert.Equal(t, m.initial, State("initial"))
	assert.True(t, m.Allowable(m.State(), State("processing")))
	assert.False(t, m.Allowable(m.State(), State("finished")))
	assert.NoError(t, m.Transition(State("processing")))
	assert.Error(t, m.Transition(State("initial")))
	assert.Equal(t, m.current, State("processing"))
	assert.NoError(t, m.Transition("finished"))
}

func TestMachineStop(t *testing.T) {
	m, err := NewMachine(State("initial"), WithStoppable(), WithTransitions(
		T(State("initial"), State("processing")),
		T(State("processing"), State("error"), State("finished")),
	))
	assert.NoError(t, err)
	assert.True(t, m.stoppable.stopOnError)
	assert.NoError(t, m.Transition(State("processing")))
	assert.Error(t, m.Transition(State("initial")))
	// after illegal transition should be stopped
	assert.True(t, m.stoppable.stopped)
	assert.Equal(t, m.current, State("processing"))
	// even allowable transitions should not be allowed after stop
	assert.Error(t, m.Transition(State("finished")))
	m.Reset()
	// should reset stopped and set to initial
	assert.False(t, m.stoppable.stopped)
	assert.Equal(t, m.current, m.initial)
	assert.True(t, m.stoppable.stopOnError)
}
