package monny

import (
	"testing"

	"github.com/stretchr/testify/mock"
)

// test helper silences superfluous logging calls from the mock package
type foo struct {
	t *testing.T
}

func (f foo) Logf(format string, args ...interface{}) {
	// makes mock calls to log a no op to prevent a lot of superfluous logging calls
}
func (f foo) Errorf(format string, args ...interface{}) {
	f.t.Errorf(format, args...)
}
func (f foo) FailNow() {
	f.t.FailNow()
}

func silenceT(t *testing.T) mock.TestingT {
	return foo{t}
}
