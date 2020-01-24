package proc

import (
	"reflect"

	"github.com/BTBurke/monny/pkg/eventbus"
)

type SinkError struct {
	err error
}

func (s SinkError) Error() string {
	return s.err.Error()
}

type ScanError struct {
	err error
}

func (s ScanError) Error() string {
	return s.err.Error()
}

type EventError struct {
	err error
}

func (e EventError) Error() string {
	return e.err.Error()
}

func newError(eb eventbus.EventDispatcher, e error) {
	t := reflect.TypeOf(e)
	evt, _ := eventbus.NewEvent(eventbus.EventType(t.String()), e.Error())
	eb.Dispatch(evt, eventbus.OnErrorTopic())
}
