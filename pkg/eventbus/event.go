package eventbus

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
)

// EventType represents the type of event being passed on the bus.  It allows handlers receiving the event to
// to properly unmarshal the data or decide if processing is required.
type EventType string

// Event is passed on the event bus to every subscriber on the channel. Data uses gob encoding as a wire format.  To decode,
// pass a pointer to the receiver type.  An error is returned if the receiver is of the wrong type to decode the data.
//type Event interface {
//	Type() EventType
//	Decode(receiver interface{}) error
//}

type Event struct {
	t EventType
	d []byte
}

// String satisfies the stringer interface
func (e Event) String() string {
	return string(e.t)
}

func (e Event) Type() EventType { return e.t }
func (e Event) Decode(receiver interface{}) error {
	dec := gob.NewDecoder(bytes.NewReader(e.d))
	if err := dec.Decode(receiver); err != nil {
		return fmt.Errorf("failed to decode event %s: %v", e.t, err)
	}
	return nil
}

func NewEvent(t EventType, data interface{}) (Event, error) {
	var b bytes.Buffer
	if data != nil {
		gob.Register(data)
		enc := gob.NewEncoder(&b)
		if err := enc.Encode(data); err != nil {
			return Event{}, fmt.Errorf("failed to create event %s: %v", t, err)
		}
	}

	log.Printf("encoded: %v", string(b.Bytes()))
	return Event{t: t, d: b.Bytes()}, nil
}

func NewErrorEvent(t EventType, err error) (Event, error) {
	return NewEvent(t, err)
}
