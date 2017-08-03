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
	"github.com/cenkalti/backoff"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func reportFromCommand(c *Command) pb.Report {
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
		ReportReason:  pb.ReportReason(c.ReportReason),
		Start:         c.Start.Unix(),
		Finish:        c.Finish.Unix(),
		Duration:      c.Duration.String(),
		ExitCode:      c.ExitCode,
		ExitCodeValid: c.ExitCodeValid,
		Messages:      c.Messages,
		Matches:       marshalMatches(c.AlertMatches),
		UserCommand:   strings.Join(c.UserCommand, " "),
		Config:        marshalConfig(c.Config),
		CreatedAt:     time.Now().Unix(),
	}
}

type report struct {
	host   string
	port   string
	report pb.Report
	opts   []grpc.DialOption
}

func NewReport(c *Command, opts ...grpc.DialOption) *report {
	r := report{
		host:   c.Config.host,
		port:   c.Config.port,
		report: reportFromCommand(c),
		opts:   opts,
	}
	if c.Config.useTLS {
		r.opts = append(r.opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	}
	return &r
}

func (r *report) Send(result chan error, cancel chan bool) {
	send := func() error {
		conn, err := grpc.Dial(net.JoinHostPort(r.host, r.port), r.opts...)
		if err != nil {
			return err
		}
		defer conn.Close()

		client := pb.NewReportsClient(conn)
		ack, err := client.Create(context.Background(), &r.report)
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

func marshalMatches(a []AlertMatch) []byte {
	// ignore error here because the report
	// should be sent even if this conversion fails
	b, _ := json.Marshal(a)
	return b
}

func marshalCreated(a []File) []byte {
	// ignore error here because the report
	// should be sent even if this conversion fails
	b, _ := json.Marshal(a)
	return b
}

func marshalConfig(a Config) []byte {
	// ignore error here because the report
	// should be sent even if this conversion fails
	b, _ := json.Marshal(a)
	return b
}
