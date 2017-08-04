package monitor

import (
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfigOptions(t *testing.T) {
	assert := assert.New(t)

	tt := []struct {
		Name   string
		Option ConfigOption
		Expect Config
		Error  bool
	}{
		{Name: "ID", Option: ID("test"), Expect: Config{ID: "test"}},
		{Name: "rule valid regex", Option: Rule(".*"), Expect: Config{Rules: []rule{rule{Regex: regexp.MustCompile(".*")}}}},
		{Name: "rule invalid regex", Option: Rule("("), Error: true},
		{Name: "JSON rule valid regex", Option: JSONRule("test", ".*"), Expect: Config{Rules: []rule{rule{Field: "test", Regex: regexp.MustCompile(".*")}}}},
		{Name: "JSON rule invalid regex", Option: JSONRule("test", "("), Error: true},
		{Name: "rule quantity", Option: RuleQuantity("5"), Expect: Config{RuleQuantity: 5}},
		{Name: "rule quantity non-numeric", Option: RuleQuantity("A"), Error: true},
		{Name: "rule period", Option: RulePeriod("2h"), Expect: Config{RulePeriod: time.Duration(2 * time.Hour)}},
		{Name: "rule period non-duration", Option: RulePeriod("2a"), Error: true},
		{Name: "stdout history", Option: StdoutHistory("50"), Expect: Config{StdoutHistory: 50}},
		{Name: "stdout history non-numeric", Option: StdoutHistory("2a"), Error: true},
		{Name: "stderr history", Option: StderrHistory("50"), Expect: Config{StderrHistory: 50}},
		{Name: "stderr history non-numeric", Option: StderrHistory("2a"), Error: true},
		{Name: "no notify on success", Option: NoNotifyOnSuccess(), Expect: Config{NotifyOnSuccess: false}},
		{Name: "no notify on failure", Option: NoNotifyOnFailure(), Expect: Config{NotifyOnFailure: false}},
		{Name: "daemon", Option: Daemon(), Expect: Config{Daemon: true}},
		{Name: "memory warn GB", Option: MemoryWarn("2G"), Expect: Config{MemoryWarn: 2000000}},
		{Name: "memory warn MB", Option: MemoryWarn("2M"), Expect: Config{MemoryWarn: 2000}},
		{Name: "memory warn KB", Option: MemoryWarn("2K"), Expect: Config{MemoryWarn: 2}},
		{Name: "memory warn invalid", Option: MemoryWarn("2T"), Error: true},
		{Name: "memory kill GB", Option: MemoryKill("2G"), Expect: Config{MemoryKill: 2000000}},
		{Name: "memory kill MB", Option: MemoryKill("2M"), Expect: Config{MemoryKill: 2000}},
		{Name: "memory kill KB", Option: MemoryKill("2K"), Expect: Config{MemoryKill: 2}},
		{Name: "memory kill invalid", Option: MemoryKill("2T"), Error: true},
		{Name: "timeout kill", Option: KillTimeout("2h"), Expect: Config{KillTimeout: time.Duration(2 * time.Hour)}},
		{Name: "timeout kill invalid", Option: KillTimeout("2T"), Error: true},
		{Name: "timeout warn", Option: NotifyTimeout("2h"), Expect: Config{NotifyTimeout: time.Duration(2 * time.Hour)}},
		{Name: "timeout warrn invalid", Option: NotifyTimeout("2T"), Error: true},
		{Name: "creates", Option: Creates("/path/to/something"), Expect: Config{Creates: []string{"/path/to/something"}}},
		{Name: "host", Option: Host("test.com:443"), Expect: Config{host: "test.com", port: "443"}},
		{Name: "host invalid", Option: Host("test.com"), Error: true},
		{Name: "insecure", Option: Insecure(), Expect: Config{useTLS: false}},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			c := Config{}
			err := tc.Option(&c)
			if !tc.Error {
				assert.Equal(tc.Expect, c)
			}
			switch tc.Error {
			case false:
				assert.NoError(err, "unexpected error in option %s", tc.Name)
			default:
				assert.Error(err, "expected error in %s", tc.Name)
			}
		})
	}

	t.Run("suppress error reporting", func(t *testing.T) {
		c := Config{}
		f := NoErrorReports()
		err := f(&c)
		assert.NoError(err)
		assert.Equal(true, SuppressErrorReporting)
	})
}

func TestConfigConstruction(t *testing.T) {
	host, _ := os.Hostname()
	tt := []struct {
		Name    string
		Options []ConfigOption
		Expect  Config
		Error   bool
	}{
		{Name: "only ID", Options: []ConfigOption{ID("test")}, Expect: Config{
			ID:              "test",
			StdoutHistory:   30,
			StderrHistory:   30,
			NotifyOnSuccess: true,
			NotifyOnFailure: true,
			Hostname:        host,
			host:            api,
			port:            port,
			useTLS:          true,
		}},
		{Name: "multiple option", Options: []ConfigOption{ID("test"), Insecure()}, Expect: Config{
			ID:              "test",
			StdoutHistory:   30,
			StderrHistory:   30,
			NotifyOnSuccess: true,
			NotifyOnFailure: true,
			Hostname:        host,
			host:            api,
			port:            port,
			useTLS:          false,
		}},
		{Name: "no ID", Options: []ConfigOption{}, Error: true},
		{Name: "option error", Options: []ConfigOption{ID("test"), Rule("(")}, Error: true},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			c, err := newConfig(tc.Options...)

			switch tc.Error {
			case false:
				assert.Equal(t, tc.Expect, c)
				assert.Equal(t, 0, len(err))
			default:
				assert.NotZero(t, len(err))
				assert.Error(t, err[0])
			}
		})
	}
}
