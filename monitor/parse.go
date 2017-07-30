package monitor

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/go-yaml/yaml"
	"github.com/spf13/pflag"
)

func init() {
	pflag.StringP("config", "c", "", "Use yaml configuration file")
	pflag.String("alert", "", "Creates a notification if this string appears in the output.  Regex OK.")
	pflag.String("alert-json", "", "Creates a notification if this text appears in the JSON output.  Accepts the field and a regular expression or simple text separated by a colon (e.g. field:value).  Nested JSON structures are accessed using a flattened path with a dot (e.g. field.nested:value).")
	pflag.Int("stdout-history", 50, "Number of lines of stdout to send with the report.")
	pflag.Int("stderr-history", 50, "Number of lines of stderr to send with the report.")
	pflag.Bool("no-notify-on-success", false, "Do not send a report on succesful completion of this process.")
	pflag.Bool("no-notify-on-failure", false, "Do not send a notification on failure.")
	pflag.Bool("daemon", false, "Designate this process as a daemon or long-running process. Any notifications triggered will be sent immediately instead of waiting for the process to finish.")
	pflag.String("memory-warn", "", "Send a notification when memory use exceeds the value.  Accepts integers ending in K, M, G.  Example: 100M")
	pflag.String("memory-kill", "", "Kill the process and send a notification when memory use exceeds the value.  Accepts integers ending in K, M, G.  Example: 100M")
	pflag.Duration("timeout-warn", time.Duration(0), "Send a notification if process duration exceeds value (e.g., 32m).  Accepts values in us, s, m, h.")
	pflag.Duration("timeout-kill", time.Duration(0), "Kill process and send a notification if process duration exceeds value (e.g., 32m).  Accepts values in us, s, m, h.")
	pflag.String("creates", "", "Send notification if file is not created after end of process")
}

type options struct {
	options []ConfigOption
}

func ParseCommandLine() {
	options := options{}
	pflag.ParseAll(parseFlag(&options))
	fmt.Printf("Found %d options", len(options.options))
}

func parseFlag(o *options) func(*pflag.Flag, string) error {
	return func(flag *pflag.Flag, value string) error {
		fmt.Printf("Flag: %s Value: %s", flag.Name, value)
		switch flag.Name {
		case "config":
			opts, err := parseFromFile(value)
			if err != nil {
				return err
			}
			o.options = append(o.options, opts...)
		default:
			option, err := handleOption(flag.Name, value)
			if err != nil {
				return err
			}
			o.options = append(o.options, option)
		}
		return nil
	}
}

func handleOption(name string, value string) (ConfigOption, error) {
	switch name {
	case "alert":
		return Alert(value), nil
	case "alert-json":
		jalert := strings.SplitAfterN(value, ":", 2)
		if len(jalert) != 2 {
			return nil, fmt.Errorf("invalid format for json alert, should be field:value only in %s", value)
		}
		return JSONAlert(jalert[0], jalert[1]), nil
	case "stdout-history":
		return StdoutHistory(value), nil
	case "stderr-history":
		return StderrHistory(value), nil
	case "no-notify-on-success":
		return NoNotifyOnSuccess(), nil
	case "no-notify-on-failure":
		return NoNotifyOnFailure(), nil
	case "daemon":
		return Daemon(), nil
	case "memory-warn":
		return MemoryWarn(value), nil
	case "memory-kill":
		return MemoryKill(value), nil
	case "timeout-warn":
		return NotifyTimeout(value), nil
	case "timeout-kill":
		return KillTimeout(value), nil
	case "creates":
		return Creates(value), nil
	default:
		return nil, fmt.Errorf("unknown option: %s", name)
	}
}

func parseFromFile(fpath string) ([]ConfigOption, error) {
	var options []ConfigOption
	data, err := ioutil.ReadFile(fpath)
	if err != nil {
		return options, err
	}

	cfg := make(map[string]interface{})
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return options, err
	}
	for k, v := range cfg {
		var opt ConfigOption
		var err error
		switch v.(type) {
		case string:
			opt, err = handleOption(k, v.(string))
			if err != nil {
				return options, nil
			}
			options = append(options, opt)
		case int:
			opt, err := handleOption(k, strconv.Itoa(v.(int)))
			if err != nil {
				return options, nil
			}
			options = append(options, opt)
		case bool:
			opt, err := handleOption(k, "")
			if err != nil {
				return options, nil
			}
			options = append(options, opt)
		// handles the case of a list of alerts
		case interface{}:
			alt := alertYAML{}
			if err := yaml.Unmarshal(data, &alt); err != nil {
				return options, fmt.Errorf("Could not unmarshal config value for key: %s", k)
			}
			for _, val := range alt.Alert {
				opt, err := handleOption("alert", val)
				if err != nil {
					return options, nil
				}
				options = append(options, opt)
			}
			for _, val := range alt.JSONAlert {
				opt, err := handleOption("alert", val)
				if err != nil {
					return options, nil
				}
				options = append(options, opt)
			}
		default:
			return options, fmt.Errorf("Could not process config key %s, unknown type", k)
		}
	}
	return options, nil
}

type alertYAML struct {
	Alert     []string `yaml:"alert"`
	JSONAlert []string `yaml:"alert-json"`
}
