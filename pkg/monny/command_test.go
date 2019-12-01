package monny

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/BTBurke/monny/pkg/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockHandlers struct {
	mock.Mock
}

func (m mockHandlers) Finished(c *Command, cmd *exec.Cmd) error {
	args := m.Called()
	return args.Error(0)
}

func (m mockHandlers) Signal(c *Command, cmd *exec.Cmd, sig os.Signal) error {
	args := m.Called()
	return args.Error(0)
}

func (m mockHandlers) Timeout(c *Command, cmd *exec.Cmd) error {
	cmd.Process.Kill()
	args := m.Called()
	return args.Error(0)
}

func (m mockHandlers) TimeWarning(c *Command) error {
	args := m.Called()
	return args.Error(0)
}

func (m mockHandlers) CheckMemory(c *Command, cmd *exec.Cmd) error {
	args := m.Called()
	return args.Error(0)
}

func (m mockHandlers) KillOnHighMemory(c *Command, cmd *exec.Cmd) error {
	cmd.Process.Kill()
	args := m.Called()
	return args.Error(0)
}

type mockReport struct{}

func (m *mockReport) Send(c *Command, reason proto.ReportReason) {
	return
}

func (m *mockReport) Wait() error {
	return nil
}

func TestHandlerCalls(t *testing.T) {
	tt := []struct {
		Name     string
		Cmd      string
		Handlers []string
		Options  []ConfigOption
		Error    []error
	}{
		{Name: "finished success", Cmd: "echo test", Handlers: []string{"Finished"}, Error: []error{nil}},
		{Name: "finished fail", Cmd: "exit(-1)", Handlers: []string{"Finished"}, Error: []error{nil}},
		{Name: "mem check", Cmd: "sleep 1", Handlers: []string{"CheckMemory", "Finished"}, Error: []error{nil, nil}},
		{Name: "mem kill", Cmd: "sleep 5", Options: []ConfigOption{MemoryKill("1K")}, Handlers: []string{"CheckMemory", "KillOnHighMemory"}, Error: []error{fmt.Errorf("high mem kill"), nil}},
		{Name: "time warning", Cmd: "sleep 1", Options: []ConfigOption{NotifyTimeout("200ms")}, Handlers: []string{"CheckMemory", "Finished", "TimeWarning"}, Error: []error{nil, nil, nil}},
		{Name: "time kill", Cmd: "sleep 1", Options: []ConfigOption{KillTimeout("200ms")}, Handlers: []string{"Timeout"}, Error: []error{nil}},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			r, w := io.Pipe()
			go func() {
				buf := new(bytes.Buffer)
				buf.ReadFrom(r)
				r.Close()
			}()
			opts := append(tc.Options, ID("test"), logOut(w), logErr(w))
			cfg, err := newConfig(opts...)
			if err != nil {
				t.Fatalf("unexpected error in config: %s", err)
			}
			mocks := new(mockHandlers)
			mockRpt := new(mockReport)

			c := &Command{
				Config:      cfg,
				UserCommand: strings.Split(tc.Cmd, " "),
				handler:     mocks,
				// Report calls not tested here, mocked only to prevent external calls
				report: mockRpt,
				err:    w,
				out:    w,
			}

			for idx, handler := range tc.Handlers {
				mocks.On(handler).Return(tc.Error[idx])
			}

			if err := c.Exec(); err != nil {
				t.Fatalf("unexpected run error: %s", err)
			}
			if err := c.Cleanup(); err != nil {
				t.Fatalf("unexpected cleanup error: %s", err)
			}

			mocks.AssertExpectations(silenceT(t))
		})
	}
}

func TestIntegration(t *testing.T) {
	tt := []struct {
		Name         string
		Cmd          string
		Stdout       []string
		Stderr       []string
		Options      []ConfigOption
		ReportReason proto.ReportReason
		KillReason   proto.KillReason
		Duration     time.Duration
		Cleanup      func()
	}{
		{Name: "capture stdout", Cmd: "echo start && sleep 1 && echo end", Stdout: []string{"start", "end"}, ReportReason: proto.Success, Duration: time.Duration(1 * time.Second)},
		{Name: "get failure exit code", Cmd: "exit(-1)", ReportReason: proto.Failure},
		{Name: "kill on timeout", Cmd: "sleep 3", Options: []ConfigOption{KillTimeout("200ms")}, ReportReason: proto.Killed, KillReason: proto.Timeout, Duration: time.Duration(200 * time.Millisecond)},
		{Name: "kill on memory", Cmd: "sleep 3", Options: []ConfigOption{MemoryKill("1K")}, ReportReason: proto.Killed, KillReason: proto.Memory},
		{Name: "file creation success", Cmd: "touch testfile.test", Options: []ConfigOption{Creates("testfile.test")}, ReportReason: proto.Success, Cleanup: func() { os.Remove("testfile.test") }},
		{Name: "file creation failed", Cmd: "touch testfile1.test", Options: []ConfigOption{Creates("testfile.test")}, ReportReason: proto.FileNotCreated, Cleanup: func() { os.Remove("testfile1.test") }},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			r, w := io.Pipe()
			go func() {
				buf := new(bytes.Buffer)
				buf.ReadFrom(r)
				r.Close()
			}()
			opts := append(tc.Options, ID("test"), logErr(w), logOut(w))
			c, err := New(strings.Split(tc.Cmd, " "), opts...)
			if err != nil {
				t.Fatalf("unexpected error setting config: %s", err)
			}
			c.report = new(mockReport)
			if err := c.Exec(); err != nil {
				t.Fatalf("unexpected error execing command: %s", err)
			}
			if err := c.Cleanup(); err != nil {
				t.Fatalf("unexpected cleanup error: %s", err)
			}
			if len(tc.Stdout) > 0 {
				assert.Equal(t, tc.Stdout, c.Stdout)
			}
			assert.Equal(t, tc.ReportReason, c.ReportReason)
			if tc.Duration > 0 {
				assert.Condition(t, duration(tc.Duration, c.Duration, 500))
			}
			if tc.Cleanup != nil {
				tc.Cleanup()
			}

		})
	}
}

func duration(expected time.Duration, actual time.Duration, deltaMillis float64) assert.Comparison {
	return func() bool {
		return math.Abs(float64(expected)-float64(actual)) < (deltaMillis * 1000000)
	}
}

const testJSON string = `{"code": 404,"msg": "test message","array": ["test1", "test2", "test3"],"nested": {"nest1": "test"},"bool": true}`

func TestExtractJSON(t *testing.T) {
	tt := []struct {
		Name   string
		Field  string
		Expect string
	}{
		{Name: "string", Field: "msg", Expect: "test message"},
		{Name: "int coerced to JSON float", Field: "code", Expect: fmt.Sprintf("%f", float64(404))},
		{Name: "nested string", Field: "nested.nest1", Expect: "test"},
		{Name: "nested array", Field: "array", Expect: "test1\ntest2\ntest3"},
		{Name: "bool", Field: "bool", Expect: "true"},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			ext := extractTextFromJSON([]byte(testJSON), tc.Field)
			assert.Equal(t, tc.Expect, string(ext))
		})
	}
}

func TestCheckRule(t *testing.T) {
	tt := []struct {
		Name  string
		Line  string
		Field string
		Regex string
		Match bool
	}{
		{Name: "text", Line: "this is a test line", Field: "", Regex: "te.*", Match: true},
		{Name: "text with capture", Line: "this is a test line", Field: "", Regex: "(te.*) line", Match: true},
		{Name: "text no match", Line: "this is a test line", Field: "", Regex: "ti.*", Match: false},
		{Name: "json text no match", Line: testJSON, Field: "msg", Regex: "ti.*", Match: false},
		{Name: "json text match", Line: testJSON, Field: "msg", Regex: "te.*", Match: true},
		{Name: "json text nested", Line: testJSON, Field: "nested.nest1", Regex: "te.*", Match: true},
		{Name: "json array", Line: testJSON, Field: "array", Regex: "te.*", Match: true},
		{Name: "json number", Line: testJSON, Field: "code", Regex: "404", Match: true},
	}
	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			reg := regexp.MustCompile(tc.Regex)
			r := rule{
				Field: tc.Field,
				Regex: reg,
			}

			matches := checkRule([]byte(tc.Line), []rule{r})
			switch tc.Match {
			case true:
				assert.Len(t, matches, 1)
			default:
				assert.Len(t, matches, 0)
			}
		})
	}
}

func TestRules(t *testing.T) {
	tt := []struct {
		Name        string
		Stdout      []string
		Options     []ConfigOption
		ExpectMatch []bool
	}{
		{Name: "simple match", Stdout: []string{"this is a test string"}, Options: []ConfigOption{Rule("test")}, ExpectMatch: []bool{true}},
		{Name: "regex match", Stdout: []string{"this is a test string"}, Options: []ConfigOption{Rule("te.*")}, ExpectMatch: []bool{true}},
		{Name: "regex match multiline", Stdout: []string{"this is a test string", "this is ten"}, Options: []ConfigOption{Rule("te.*")}, ExpectMatch: []bool{true, true}},
		{Name: "json regex match", Stdout: []string{testJSON}, Options: []ConfigOption{JSONRule("msg", "te.*")}, ExpectMatch: []bool{true}},
		{Name: "json simple match", Stdout: []string{testJSON}, Options: []ConfigOption{JSONRule("code", "404")}, ExpectMatch: []bool{true}},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			r, w := io.Pipe()
			go func() {
				buf := new(bytes.Buffer)
				buf.ReadFrom(r)
				r.Close()
			}()
			opts := append(tc.Options, ID("test"), logErr(w), logOut(w))
			c, err := New([]string{"echo", "\"" + strings.Replace(strings.Join(tc.Stdout, "\n"), "\"", "\\\"", -1) + "\""}, opts...)
			if err != nil {
				t.Fatalf("unexpected error in config: %s", err)
			}
			c.report = new(mockReport)

			if err := c.Exec(); err != nil {
				t.Fatalf("unexpected error running: %s", err)
			}
			if err := c.Cleanup(); err != nil {
				t.Fatalf("unexpected error cleaning up: %s", err)
			}
			for idx, em := range tc.ExpectMatch {
				switch em {
				case true:
					if len(c.RuleMatches) <= idx {
						t.Fatalf("no rule match exists, expected match for test case %s for:\n%s", tc.Name, tc.Stdout[idx])
					}
					assert.Equal(t, tc.Stdout[idx], c.RuleMatches[idx].Line)
				default:
					continue
				}
			}
		})
	}

}
