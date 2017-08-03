package monitor

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/BTBurke/wtf/pb"
	"github.com/BTBurke/wtf/proto"
	"github.com/cenkalti/backoff"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// ReportSender is an interface for sending reports
type ReportSender interface {
	Create(c *Command, reason proto.ReportReason, opts ...grpc.DialOption)
	Send(result chan error, cancel chan bool)
}

// reportFromCommand converts a Command to a pb.Report, doing
// some conversion to be compatible with PB types and storage
// schema on the backend
func reportFromCommand(c *Command, reason proto.ReportReason) pb.Report {
	return pb.Report{
		Id:            c.Config.ID,
		Hostname:      c.Config.Hostname,
		Stdout:        c.Stdout,
		Stderr:        c.Stderr,
		Success:       c.Success,
		MaxMemory:     c.MaxMemory,
		Killed:        c.Killed,
		KillReason:    pb.KillReason(c.KillReason),
		Created:       marshalCreated(c.Created),
		ReportReason:  pb.ReportReason(reason),
		Start:         c.Start.Unix(),
		Finish:        c.Finish.Unix(),
		Duration:      c.Duration.String(),
		ExitCode:      c.ExitCode,
		ExitCodeValid: c.ExitCodeValid,
		Messages:      c.Messages,
		Matches:       marshalMatches(c.RuleMatches),
		UserCommand:   strings.Join(c.UserCommand, " "),
		Config:        marshalConfig(c.Config),
		CreatedAt:     time.Now().Unix(),
	}
}

// Report is a wrapper for sending a report via GRPC. See pb.Report for details.
type Report struct {
	Host  string
	Port  string
	Proto pb.Report
	Opts  []grpc.DialOption
}

// Create prepares a new report based on the current status of the command.
func (r *Report) Create(c *Command, reason proto.ReportReason, opts ...grpc.DialOption) {
	r.Proto = reportFromCommand(c, reason)
	if c.Config.useTLS {
		r.Opts = append(r.Opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	}
	return
}

// Send will transmit a report to the notification server using a go routine.
// Errors will cause an exponential backoff until the call is successful or a timeout
// is received from the parent.
func (r *Report) Send(result chan error, cancel chan bool) {
	send := func() error {
		conn, err := grpc.Dial(net.JoinHostPort(r.Host, r.Port), r.Opts...)
		if err != nil {
			return err
		}
		defer conn.Close()

		client := pb.NewReportsClient(conn)
		ack, err := client.Create(context.Background(), &r.Proto)
		if err != nil {
			return err
		}
		if !ack.Success {
			return fmt.Errorf("send fail")
		}
		return nil
	}
	select {
	case result <- backoff.Retry(send, backoff.NewExponentialBackOff()):
	case <-cancel:
	}
}

func marshalMatches(a []RuleMatch) []byte {
	b, err := json.Marshal(a)
	if err != nil {
		// Error will be reported externally, but should
		// not happen.  Report will continue even if this
		// conversion fails.
		ReportError(err)
	}
	return b
}

func marshalCreated(a []File) []byte {
	b, err := json.Marshal(a)
	if err != nil {
		// Error will be reported externally, but should
		// not happen.  Report will continue even if this
		// conversion fails.
		ReportError(err)
	}
	return b
}

func marshalConfig(a Config) []byte {
	b, err := json.Marshal(a)
	if err != nil {
		// Error will be reported externally, but should
		// not happen.  Report will continue even if this
		// conversion fails.
		ReportError(err)
	}
	return b
}
