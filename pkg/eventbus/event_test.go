package eventbus

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Tester struct {
	A int
	B string
}

func TestStructEvent(t *testing.T) {
	e := EventType("test_event")

	tt := []struct {
		name string
		in   Tester
	}{
		{name: "basic", in: Tester{A: 999, B: "test"}},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			evt, err := NewEvent(e, tc.in)
			assert.NoError(t, err)
			var r Tester
			errDec := evt.Decode(&r)
			assert.NoError(t, errDec)
			assert.Equal(t, tc.in, r)
		})
	}
}

func TestNilData(t *testing.T) {
	_, err := NewEvent(EventType("test_event"), nil)
	assert.NoError(t, err)
}

func TestPrimitiveTypes(t *testing.T) {
	evt, err := NewEvent(EventType("test"), []byte("test string"))
	assert.NoError(t, err)
	var b []byte
	errD := evt.Decode(&b)
	assert.NoError(t, errD)
	assert.Equal(t, []byte("test string"), b)
}

func TestErrorEvent(t *testing.T) {
	e := fmt.Errorf("test error")
	evt, err := NewErrorEvent(EventType("test_error"), e)
	assert.NoError(t, err)
	var e2 error
	errD := evt.Decode(&e2)
	assert.NoError(t, errD)
	assert.Equal(t, e, e2)
}
