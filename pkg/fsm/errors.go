package fsm

// TransitionNotAllowed is an error type caused by attempting to transition to a state that is
// not allowed by the FSM
type TransitionNotAllowed struct {
	Msg string
}

func (e TransitionNotAllowed) Error() string {
	return e.Msg
}

// StopError is thrown when a state machine is in a stopped state due to an unallowable transition
type StopError struct {
	Msg string
}

func (e StopError) Error() string {
	return e.Msg
}

// NonceError is thrown when a nonce-enabled state machine attempts to transition
// with an incorrect nonce
type NonceError struct {
	Msg string
}

func (e NonceError) Error() string {
	return e.Msg
}
