package eventbus

// EventType represents the type of event being passed on the bus.  It allows handlers receiving the event to
// to properly unmarshal the data or decide if processing is required
type EventType string

// Event is passed on the event bus to every subscriber on the channel
type Event struct {
	EventType EventType
	Data      interface{}
}
