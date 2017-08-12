package monitor

import (
	"time"

	"github.com/BTBurke/wtf/pb"
	"github.com/BTBurke/wtf/proto"

	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockSender struct {
	mock.Mock
}

func (m *mockSender) create(c *Command, reason proto.ReportReason) *pb.Report {
	args := m.Called()
	return args.Get(0).(*pb.Report)
}
func (m *mockSender) sendBackground(report *pb.Report, result chan error, cancel chan bool) {
	m.Called()
	result <- nil
}

func TestReportCreation(t *testing.T) {
	tt := []struct {
		Name       string
		ShouldSend bool
		Reason     proto.ReportReason
		TestCase   func() (*Command, *Command)
	}{
		{Name: "success", ShouldSend: true, Reason: proto.Success, TestCase: baseCase(proto.Success)},
		{Name: "success no-report-success", ShouldSend: false, Reason: proto.Success, TestCase: baseCase(proto.Success, NoNotifyOnSuccess())},
		{Name: "failure", ShouldSend: true, Reason: proto.Failure, TestCase: baseCase(proto.Failure)},
		{Name: "failure no-report-failure", ShouldSend: false, Reason: proto.Failure, TestCase: baseCase(proto.Failure, NoNotifyOnFailure())},
		{Name: "alert", ShouldSend: true, Reason: proto.Alert, TestCase: alertCase(true)},
		{Name: "alert rate exceed no duration", ShouldSend: true, Reason: proto.AlertRate, TestCase: alertCase(true, RuleQuantity("5"))},
		{Name: "alert rate exceed duration", ShouldSend: true, Reason: proto.AlertRate, TestCase: alertCase(true, RuleQuantity("5"), RulePeriod("1h"))},
		{Name: "alert rate under", ShouldSend: false, Reason: proto.AlertRate, TestCase: alertCase(false, RuleQuantity("5"), RulePeriod("1h"))},
		{Name: "killed", ShouldSend: true, Reason: proto.FileNotCreated, TestCase: baseCase(proto.FileNotCreated)},
		{Name: "file not created", ShouldSend: true, Reason: proto.Killed, TestCase: baseCase(proto.Killed)},
		{Name: "start daemon", ShouldSend: true, Reason: proto.Start, TestCase: baseCase(proto.Start, Daemon())},
		{Name: "start no daemon", ShouldSend: false, Reason: proto.Start, TestCase: baseCase(proto.Start)},
		{Name: "warn time", ShouldSend: true, Reason: proto.TimeWarning, TestCase: baseCase(proto.TimeWarning)},
		{Name: "warn memory", ShouldSend: true, Reason: proto.MemoryWarning, TestCase: baseCase(proto.MemoryWarning)},
	}

	for _, tc := range tt {

		mocks := new(mockSender)
		r := &Report{
			sender: mocks,
		}

		testConfig, expectConfig := tc.TestCase()
		mocks.On("create").Return(reportFromCommand(testConfig, tc.Reason, nil))
		if tc.ShouldSend {
			mocks.On("sendBackground")
		}

		r.Send(testConfig, tc.Reason)

		mocks.AssertExpectations(t)
		assert.EqualValues(t, expectConfig, testConfig)
	}
}

func baseCase(reason proto.ReportReason, opts ...ConfigOption) func() (*Command, *Command) {
	return func() (*Command, *Command) {
		opts = append(opts, ID("test"))
		cmd, _ := New([]string{"test"}, opts...)
		cmd.ReportReason = reason
		return cmd, cmd
	}
}

func alertCase(exceed bool, opts ...ConfigOption) func() (*Command, *Command) {
	return func() (*Command, *Command) {
		opts = append(opts, ID("test"))
		cmd, _ := New([]string{"test"}, opts...)

		var rm []RuleMatch
		if exceed {
			rm = createMatches(cmd.Config.RulePeriod, 2*cmd.Config.RuleQuantity)
		} else {
			rm = createMatches(2*cmd.Config.RulePeriod, cmd.Config.RuleQuantity)
		}
		cmd.RuleMatches = rm

		cmdReturn := &Command{}
		*cmdReturn = *cmd
		// tests whether rule matches are cleared after send
		if exceed {
			cmdReturn.RuleMatches = []RuleMatch{}
		}

		return cmd, cmdReturn
	}
}

func TestRateCheck(t *testing.T) {
	tt := []struct {
		Name        string
		Quantity    int
		Duration    time.Duration
		RuleMatches []RuleMatch
		Exceeds     bool
	}{
		{Name: "exceeds", Quantity: 3, Duration: time.Duration(1 * time.Minute), RuleMatches: createMatches(time.Duration(30*time.Second), 6), Exceeds: true},
		{Name: "no exceed", Quantity: 3, Duration: time.Duration(1 * time.Minute), RuleMatches: createMatches(time.Duration(30*time.Second), 1), Exceeds: false},
		{Name: "no duration exceeds", Quantity: 3, Duration: time.Duration(0), RuleMatches: createMatches(time.Duration(0), 4), Exceeds: true},
		{Name: "no duration", Quantity: 3, Duration: time.Duration(0), RuleMatches: createMatches(time.Duration(0), 2), Exceeds: false},
		{Name: "slow rate", Quantity: 40, Duration: time.Duration(10 * time.Minute), RuleMatches: createMatches(time.Duration(20*time.Minute), 40), Exceeds: false},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			exceeds := calcAlertRate(tc.RuleMatches, tc.Quantity, tc.Duration)
			assert.Equal(t, tc.Exceeds, exceeds)
		})
	}

}

func createMatches(t time.Duration, num int) []RuleMatch {
	if num == 0 {
		return []RuleMatch{}
	}
	tBuffer := int64(t)
	tDelta := -1 * tBuffer / int64(num)
	tStart := time.Now()

	var rm []RuleMatch
	for i := 0; i < num; i++ {
		rm = append(rm, RuleMatch{
			Time: tStart.Add(time.Duration(tDelta * int64(i))),
		})
	}
	return rm
}
