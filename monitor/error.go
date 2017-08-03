package monitor

import (
	"os"

	"github.com/stvp/rollbar"
)

// SuppressErrorReporting is a global flag to prevent the client
// from sending unhandled errors to Rollbar to improve the quality
// of the service.  Data is anonymous and consists only of a stack
// trace to identify the source of the problem.
var SuppressErrorReporting bool

func init() {
	switch env := os.Getenv("environment"); env {
	case "development":
		rollbar.Environment = "development"
	default:
		rollbar.Environment = "production"
	}
	rollbar.Token = "8046af1f8781407faad15c1f86c0dccc"
}

// ReportError will send the result of an unexpected error to Rollbar
// to improve the quality of the client.  Data is anonymous.
func ReportError(err error) {
	rollbar.Error(rollbar.ERR, err)
}
