package eventbus

import "fmt"

const (
	errorTopic Topic = Topic("__errors__")
)

// ErrShutdownTimeout is returned if calling eventbus.Shutdown(ctx) causes the context to timeout before all subscribers
// have exited
var ErrShutdownTimeout error = fmt.Errorf("eventbus: context timeout or cancelled before all subscribers exited")

func OnErrorTopic() Topic {
	return errorTopic
}
