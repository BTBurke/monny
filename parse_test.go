package monny

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/go-yaml/yaml"
	"github.com/stretchr/testify/assert"
)

func TestParseFlags(t *testing.T) {
	tt := []struct {
		Name     string
		Cmdline  string
		Expected []ConfigOption
		Error    bool
	}{
		{Name: "id", Cmdline: "--id test", Expected: []ConfigOption{ID("test")}, Error: false},
		{Name: "rule", Cmdline: "--rule test", Expected: []ConfigOption{Rule("test")}, Error: false},
		{Name: "rule-json", Cmdline: "--rule-json field:test", Expected: []ConfigOption{JSONRule("field", "test")}, Error: false},
		{Name: "stdout-history", Cmdline: "--stdout-history 75", Expected: []ConfigOption{StdoutHistory("75")}, Error: false},
		{Name: "stderr-history", Cmdline: "--stderr-history 75", Expected: []ConfigOption{StderrHistory("75")}, Error: false},
		{Name: "no-notify-on-success", Cmdline: "--no-notify-on-success", Expected: []ConfigOption{NoNotifyOnSuccess()}, Error: false},
		{Name: "no-notify-on-failure", Cmdline: "--no-notify-on-failure", Expected: []ConfigOption{NoNotifyOnFailure()}, Error: false},
		{Name: "daemon", Cmdline: "--daemon", Expected: []ConfigOption{Daemon()}, Error: false},
		{Name: "memory-warn", Cmdline: "--memory-warn 100K", Expected: []ConfigOption{MemoryWarn("100K")}, Error: false},
		{Name: "memory-kill", Cmdline: "--memory-kill 1G", Expected: []ConfigOption{MemoryKill("1G")}, Error: false},
		{Name: "timeout-warn", Cmdline: "--timeout-warn 10m", Expected: []ConfigOption{NotifyTimeout("10m")}, Error: false},
		{Name: "timeout-kill", Cmdline: "--timeout-kill 30m", Expected: []ConfigOption{KillTimeout("30m")}, Error: false},
		{Name: "creates", Cmdline: "--creates /path/foo/bar", Expected: []ConfigOption{Creates("/path/foo/bar")}, Error: false},
		{Name: "creates multiple", Cmdline: "--creates /path/foo/bar --creates /this/one/too", Expected: []ConfigOption{Creates("/path/foo/bar"), Creates("/this/one/too")}, Error: false},
		{Name: "host", Cmdline: "--host localhost:8080", Expected: []ConfigOption{Host("localhost:8080")}, Error: false},
		{Name: "insecure", Cmdline: "--insecure", Expected: []ConfigOption{Insecure()}, Error: false},
		{Name: "no-error-reports", Cmdline: "--no-error-reports", Expected: []ConfigOption{NoErrorReports()}, Error: false},
		{Name: "shell", Cmdline: "--shell /usr/bin/zsh", Expected: []ConfigOption{Shell("/usr/bin/zsh")}, Error: false},
		{Name: "error on unknown flag", Cmdline: "--does-not-exist", Expected: []ConfigOption{}, Error: true},
		{Name: "multiple rules", Cmdline: "--rule test --rule foo", Expected: []ConfigOption{Rule("test"), Rule("foo")}, Error: false},
		{Name: "multiple json rules", Cmdline: "--rule-json field:test --rule-json foo:bar", Expected: []ConfigOption{JSONRule("field", "test"), JSONRule("foo", "bar")}, Error: false},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			pf := createFlagSet()
			_, options, err := parse(strings.Split(tc.Cmdline, " "), pf)
			if tc.Error {
				assert.Error(t, err)
			} else {
				expected, received := createComparisonConfigs(tc.Expected, options)
				assert.Equal(t, expected, received)
				assert.NoError(t, err)

			}
		})
	}
}

func TestParseYAML(t *testing.T) {
	tt := []struct {
		Name     string
		Yaml     map[string]interface{}
		Expected []ConfigOption
		Error    bool
	}{
		{Name: "id", Yaml: map[string]interface{}{"id": "test"}, Expected: []ConfigOption{ID("test")}, Error: false},
		{Name: "rule", Yaml: map[string]interface{}{"rule": "test"}, Expected: []ConfigOption{Rule("test")}, Error: false},
		{Name: "rule-json", Yaml: map[string]interface{}{"rule-json": "field:test"}, Expected: []ConfigOption{JSONRule("field", "test")}, Error: false},
		{Name: "stdout-history", Yaml: map[string]interface{}{"stdout-history": 75}, Expected: []ConfigOption{StdoutHistory("75")}, Error: false},
		{Name: "stderr-history", Yaml: map[string]interface{}{"stderr-history": 75}, Expected: []ConfigOption{StderrHistory("75")}, Error: false},
		{Name: "no-notify-on-success", Yaml: map[string]interface{}{"no-notify-on-success": true}, Expected: []ConfigOption{NoNotifyOnSuccess()}, Error: false},
		{Name: "no-notify-on-failure", Yaml: map[string]interface{}{"no-notify-on-failure": true}, Expected: []ConfigOption{NoNotifyOnFailure()}, Error: false},
		{Name: "daemon", Yaml: map[string]interface{}{"daemon": true}, Expected: []ConfigOption{Daemon()}, Error: false},
		{Name: "memory-warn", Yaml: map[string]interface{}{"memory-warn": "100K"}, Expected: []ConfigOption{MemoryWarn("100K")}, Error: false},
		{Name: "memory-kill", Yaml: map[string]interface{}{"memory-kill": "1G"}, Expected: []ConfigOption{MemoryKill("1G")}, Error: false},
		{Name: "timeout-warn", Yaml: map[string]interface{}{"timeout-warn": "10m"}, Expected: []ConfigOption{NotifyTimeout("10m")}, Error: false},
		{Name: "timeout-kill", Yaml: map[string]interface{}{"timeout-kill": "30m"}, Expected: []ConfigOption{KillTimeout("30m")}, Error: false},
		{Name: "creates", Yaml: map[string]interface{}{"creates": "/path/foo/bar"}, Expected: []ConfigOption{Creates("/path/foo/bar")}, Error: false},
		{Name: "creates multiple", Yaml: map[string]interface{}{"creates": []string{"/path/foo/bar", "/this/one/too"}}, Expected: []ConfigOption{Creates("/path/foo/bar"), Creates("/this/one/too")}, Error: false},
		{Name: "host", Yaml: map[string]interface{}{"host": "localhost:8080"}, Expected: []ConfigOption{Host("localhost:8080")}, Error: false},
		{Name: "insecure", Yaml: map[string]interface{}{"insecure": true}, Expected: []ConfigOption{Insecure()}, Error: false},
		{Name: "no-error-reports", Yaml: map[string]interface{}{"no-error-reports": true}, Expected: []ConfigOption{NoErrorReports()}, Error: false},
		{Name: "shell", Yaml: map[string]interface{}{"shell": "/usr/bin/zsh"}, Expected: []ConfigOption{Shell("/usr/bin/zsh")}, Error: false},
		{Name: "error on unknown flag", Yaml: map[string]interface{}{"does-not-exist": "test"}, Expected: []ConfigOption{}, Error: true},
		{Name: "multiple rules", Yaml: map[string]interface{}{"rule": []string{"test", "foo"}}, Expected: []ConfigOption{Rule("test"), Rule("foo")}, Error: false},
		{Name: "multiple json rules", Yaml: map[string]interface{}{"rule-json": []string{"field:test", "foo:bar"}}, Expected: []ConfigOption{JSONRule("field", "test"), JSONRule("foo", "bar")}, Error: false},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			f, err := ioutil.TempFile("", "xrcfg")
			if err != nil {
				t.Fatalf("unexpected error creating temp config file: %s", err)
			}
			defer os.Remove(f.Name())

			y, err := yaml.Marshal(tc.Yaml)
			if err != nil {
				t.Fatalf("unexpected error marshaling YAML: %s", err)
			}
			if _, err := f.Write(y); err != nil {
				t.Fatalf("unexpected error writing to file: %s", err)
			}
			if err := f.Close(); err != nil {
				t.Fatalf("unexpected error closing file: %s", err)
			}

			pf := createFlagSet()
			_, options, err := parse([]string{"-c", f.Name()}, pf)
			if tc.Error {
				assert.Error(t, err)
			} else {
				expected, received := createComparisonConfigs(tc.Expected, options)
				assert.Equal(t, expected, received)
				assert.NoError(t, err)

			}
		})
	}
}

func createComparisonConfigs(expected []ConfigOption, received []ConfigOption) (Config, Config) {
	expectedConfig := Config{}
	for _, eo := range expected {
		eo(&expectedConfig)
	}
	receivedConfig := Config{}
	for _, to := range received {
		to(&receivedConfig)
	}
	return expectedConfig, receivedConfig
}
