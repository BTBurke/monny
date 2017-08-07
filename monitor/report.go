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

type sender interface {
	create(c *Command, reason proto.ReportReason) *pb.Report
	sendBackground(report *pb.Report, result chan error, cancel chan bool)
}

type senderService struct {
	host   string
	port   string
	opts   []grpc.DialOption
	errors ErrorReporter
}

// Create prepares a new report based on the current status of the command.
func (s *senderService) create(c *Command, reason proto.ReportReason) *pb.Report {
	pb := s.reportFromCommand(c, reason)
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

	cb := func() { return }
	switch reason {
	case proto.FileNotCreated, proto.Failure, proto.Success, proto.Killed:
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
		}
	case proto.MemoryWarning:
		if c.memWarnSent {
			close(result)
			close(cancel)
			return
		}
		go r.sender.sendBackground(pb, result, cancel)
	case proto.TimeWarning:
		if c.timeWarnSent {
			close(result)
			close(cancel)
			return
		}
		go r.sender.sendBackground(pb, result, cancel)
	case proto.Start:
		if c.Config.Daemon {
			go r.sender.sendBackground(pb, result, cancel)
		} else {
			close(result)
			close(cancel)
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
	}
	close(result)
	close(cancel)
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

// reportFromCommand converts a Command to a pb.Report, doing
// some conversion to be compatible with PB types and storage
// schema on the backend
func (s *senderService) reportFromCommand(c *Command, reason proto.ReportReason) *pb.Report {
	return &pb.Report{
		Id:            c.Config.ID,
		Hostname:      c.Config.Hostname,
		Stdout:        c.Stdout,
		Stderr:        c.Stderr,
		Success:       c.Success,
		MaxMemory:     c.MaxMemory,
		Killed:        c.Killed,
		KillReason:    pb.KillReason(c.KillReason),
		Created:       s.marshalCreated(c.Created),
		ReportReason:  pb.ReportReason(reason),
		Start:         c.Start.Unix(),
		Finish:        c.Finish.Unix(),
		Duration:      c.Duration.String(),
		ExitCode:      c.ExitCode,
		ExitCodeValid: c.ExitCodeValid,
		Messages:      c.Messages,
		Matches:       s.marshalMatches(c.RuleMatches),
		UserCommand:   strings.Join(c.UserCommand, " "),
		Config:        s.marshalConfig(c.Config),
		CreatedAt:     time.Now().Unix(),
	}
}

func (s *senderService) marshalMatches(a []RuleMatch) []byte {
	b, err := json.Marshal(a)
	if err != nil {
		// Error will be reported externally, but should
		// not happen.  Report will continue even if this
		// conversion fails.
		s.errors.ReportError(err)
	}
	return b
}

func (s *senderService) marshalCreated(a []File) []byte {
	b, err := json.Marshal(a)
	if err != nil {
		// Error will be reported externally, but should
		// not happen.  Report will continue even if this
		// conversion fails.
		s.errors.ReportError(err)
	}
	return b
}

func (s *senderService) marshalConfig(a Config) []byte {
	b, err := json.Marshal(a)
	if err != nil {
		// Error will be reported externally, but should
		// not happen.  Report will continue even if this
		// conversion fails.
		s.errors.ReportError(err)
	}
	return b
}
