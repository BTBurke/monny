package monitor

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
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
	Send(c *Command, reason proto.ReportReason)
}

// Report is a wrapper for sending a report via GRPC. See pb.Report for details.
type Report struct {
	sender sender
}

// sender is an interface for creating and sending a report in the background.
type sender interface {
	create(c *Command, reason proto.ReportReason) *pb.Report
	sendBackground(report *pb.Report, result chan error, cancel chan bool)
}

// senderService implements the sender interface to send reports in the background using GRPC
type senderService struct {
	host    string
	port    string
	timeout time.Duration
	opts    []grpc.DialOption
	errors  ErrorReporter
}

// Create prepares a new report based on the current status of the command.
func (s *senderService) create(c *Command, reason proto.ReportReason) *pb.Report {
	pb := reportFromCommand(c, reason, s.errors.ReportError)
	if c.Config.useTLS {
		s.opts = append(s.opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	}
	return pb
}

// Send will send a report based on the current run status
// of the command.  This is safe to call in a go routine to send
// in the background.  It will attempt to send a report for 1hr
// using exponential backoff if the call fails.
func (r *Report) Send(c *Command, reason proto.ReportReason) {
	log.Printf("starting send for reason %s\n", reason)
	c.mutex.Lock()
	pb := r.sender.create(c, reason)
	c.mutex.Unlock()

	result := make(chan error, 1)
	cancel := make(chan bool, 1)
	timeout := time.After(1 * time.Hour)

	closeChannels := func() {
		close(result)
		close(cancel)
	}

	cb := func() { return }
	switch reason {
	case proto.Failure:
		if c.Config.NotifyOnFailure {
			go r.sender.sendBackground(pb, result, cancel)
		} else {
			closeChannels()
			return
		}
	case proto.Success:
		if c.Config.NotifyOnSuccess {
			go r.sender.sendBackground(pb, result, cancel)
		} else {
			closeChannels()
			return
		}
	case proto.FileNotCreated, proto.Killed:
		go r.sender.sendBackground(pb, result, cancel)
	case proto.Alert:
		go r.sender.sendBackground(pb, result, cancel)
		cb = func() {
			c.RuleMatches = []RuleMatch{}
			return
		}
	case proto.AlertRate:
		alertRateExceeded := calcAlertRate(c.RuleMatches, c.Config.RuleQuantity, c.Config.RulePeriod)
		if alertRateExceeded {
			go r.sender.sendBackground(pb, result, cancel)
			cb = func() {
				c.RuleMatches = []RuleMatch{}
				return
			}
		} else {
			closeChannels()
			return
		}
	case proto.MemoryWarning:
		if c.memWarnSent {
			closeChannels()
			return
		}
		go r.sender.sendBackground(pb, result, cancel)
	case proto.TimeWarning:
		if c.timeWarnSent {
			closeChannels()
			return
		}
		go r.sender.sendBackground(pb, result, cancel)
	case proto.Start:
		if c.Config.Daemon {
			go r.sender.sendBackground(pb, result, cancel)
		} else {
			closeChannels()
			return
		}
	default:
		return
	}

	log.Printf("enter select for %s", reason.String())
	select {
	case err := <-result:
		switch {
		case err == nil:
			cb()
		default:
			c.errors.ReportError(err)
		}
	case <-timeout:
		cancel <- true
		c.errors.ReportError(fmt.Errorf("timeout on background report send: msg=%+v", pb))
	}
	closeChannels()
}

// Send will transmit a report to the notification server using a go routine.
// Errors will cause an exponential backoff until the call is successful or a timeout
// is received from the parent.
func (s *senderService) sendBackground(report *pb.Report, result chan error, cancel chan bool) {
	if report == nil {
		result <- fmt.Errorf("no report created")
		return
	}
	send := func() error {
		conn, err := grpc.Dial(net.JoinHostPort(s.host, s.port), s.opts...)
		if err != nil {
			return err
		}
		defer conn.Close()

		client := pb.NewReportsClient(conn)
		ack, err := client.Create(context.Background(), report)
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

func calcAlertRate(matches []RuleMatch, quantity int, period time.Duration) bool {
	var matchesInPeriod int
	now := time.Now()

	switch {
	case period > 0:
		for _, match := range matches {
			if now.Sub(match.Time) <= period {
				matchesInPeriod++
			}
		}
	default:
		matchesInPeriod = len(matches)
	}

	switch {
	case matchesInPeriod >= quantity:
		return true
	default:
		return false
	}
}

// reportFromCommand converts a Command to a pb.Report, doing
// some conversion to be compatible with PB types and storage
// schema on the backend
func reportFromCommand(c *Command, reason proto.ReportReason, onError func(e error)) *pb.Report {
	return &pb.Report{
		Id:            c.Config.ID,
		Hostname:      c.Config.Hostname,
		Stdout:        c.Stdout,
		Stderr:        c.Stderr,
		Success:       c.Success,
		MaxMemory:     c.MaxMemory,
		Killed:        c.Killed,
		KillReason:    pb.KillReason(c.KillReason),
		Created:       marshalCreated(c.Created, onError),
		ReportReason:  pb.ReportReason(reason),
		Start:         c.Start.Unix(),
		Finish:        c.Finish.Unix(),
		Duration:      c.Duration.String(),
		ExitCode:      c.ExitCode,
		ExitCodeValid: c.ExitCodeValid,
		Messages:      c.Messages,
		Matches:       marshalMatches(c.RuleMatches, onError),
		UserCommand:   strings.Join(c.UserCommand, " "),
		Config:        marshalConfig(c.Config, onError),
		CreatedAt:     time.Now().Unix(),
	}
}

func marshalMatches(a []RuleMatch, onError func(e error)) []byte {
	b, err := json.Marshal(a)
	if err != nil {
		// Error will be reported externally. Report will continue even if this
		// conversion fails.
		onError(err)
	}
	return b
}

func marshalCreated(a []File, onError func(e error)) []byte {
	b, err := json.Marshal(a)
	if err != nil {
		// Error will be reported externally. Report will continue even if this
		// conversion fails.
		onError(err)
	}
	return b
}

func marshalConfig(a Config, onError func(e error)) []byte {
	b, err := json.Marshal(a)
	if err != nil {
		// Error will be reported externally. Report will continue even if this
		// conversion fails.
		onError(err)
	}
	return b
}
