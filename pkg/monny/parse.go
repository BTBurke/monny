package monny

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-yaml/yaml"
	"github.com/spf13/pflag"
)

type options struct {
	options []ConfigOption
	err     error
}

// ParseCommandLine configures the client from command line options or from
// a YAML configuration file passed with the -c flag.  Returns the user command
// and a slice of functional options that can be applied to the configuration.
func ParseCommandLine() ([]string, []ConfigOption, error) {
	pf := createFlagSet()
	return parse(os.Args[1:], pf)
}

func parse(args []string, pf *pflag.FlagSet) ([]string, []ConfigOption, error) {
	options := options{}
	if err := pf.ParseAll(args, parseFlag(&options)); err != nil {
		return pf.Args(), options.options, err
	}
	return pf.Args(), options.options, options.err
}

func createFlagSet() *pflag.FlagSet {
	pf := pflag.NewFlagSet("monny", pflag.ContinueOnError)
	pf.Usage = func() {
		fmt.Printf("Usage of monny:\nmonny -i <identifier> <options> mycommand\nmonny -i <identifier> <options> -- mycommand <mycommand-options>\n")
		fmt.Printf("\n%s", pf.FlagUsagesWrapped(10))
		fmt.Printf("\n\nFor unknown flag errors, add an empty flag separator (--) between the flags for monny and your command.  Example:\n\nmonny -i id -c config.yml -- mycommand --otherflag\n")
	}

	pf.StringP("id", "i", "", "Identifier for this monitor (required)")
	pf.StringP("config", "c", "", "Use yaml configuration file")
	pf.String("rule", "", "Creates a notification if this string appears in the output.  Regex OK.")
	pf.String("rule-json", "", "Creates a notification if this text appears in the JSON output.  Accepts the field and a regular expression or simple text separated by a colon (e.g. field:value).  Nested JSON structures are accessed using a flattened path with a dot (e.g. field.nested:value).")
	pf.Int("stdout-history", 30, "Number of lines of stdout to send with the report.")
	pf.Int("stderr-history", 30, "Number of lines of stderr to send with the report.")
	pf.Bool("no-notify-on-success", false, "Do not send a report on succesful completion of this process.")
	pf.Bool("no-notify-on-failure", false, "Do not send a notification on failure.")
	pf.Bool("daemon", false, "Designate this process as a daemon or long-running process. Any notifications triggered will be sent immediately instead of waiting for the process to finish.")
	pf.String("memory-warn", "", "Send a notification when memory use exceeds the value.  Accepts integers ending in K, M, G.  Example: 100M")
	pf.String("memory-kill", "", "Kill the process and send a notification when memory use exceeds the value.  Accepts integers ending in K, M, G.  Example: 100M")
	pf.Duration("timeout-warn", time.Duration(0), "Send a notification if process duration exceeds value (e.g., 32m).  Accepts values in us, s, m, h.")
	pf.Duration("timeout-kill", time.Duration(0), "Kill process and send a notification if process duration exceeds value (e.g., 32m).  Accepts values in us, s, m, h.")
	pf.String("creates", "", "Send notification if file is not created after end of process")
	pf.String("host", "", "Host to which to send the reports as host:port")
	pf.Bool("insecure", false, "Do not use TLS to secure connection for reports")
	pf.Bool("no-error-reports", false, "Do not send reports when there are unexpected errors in the client")
	pf.String("shell", "", "Shell to use to execute command")

	return pf
}

func parseFlag(o *options) func(*pflag.Flag, string) error {
	return func(flag *pflag.Flag, value string) error {
		switch flag.Name {
		case "config":
			opts, err := parseFromFile(value)
			if err != nil {
				o.err = err
				return err
			}
			o.options = append(o.options, opts...)
		default:
			option, err := handleOption(flag.Name, value)
			if err != nil {
				o.err = err
				return err
			}
			o.options = append(o.options, option)
		}
		return nil
	}
}

func handleOption(name string, value string) (ConfigOption, error) {
	switch name {
	case "id":
		return ID(value), nil
	case "rule":
		return Rule(value), nil
	case "rule-json":
		jrule := strings.SplitAfterN(value, ":", 2)
		if len(jrule) != 2 {
			return nil, fmt.Errorf("invalid format for json rule, should be field:value only in %s", value)
		}
		return JSONRule(jrule[0][0:len(jrule[0])-1], jrule[1]), nil
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
	case "host":
		return Host(value), nil
	case "insecure":
		return Insecure(), nil
	case "no-error-reports":
		return NoErrorReports(), nil
	case "shell":
		return Shell(value), nil
	default:
		return nil, fmt.Errorf("Unknown option: %s", name)
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

		switch v.(type) {
		case string:
			opt, err := handleOption(k, v.(string))
			if err != nil {
				return options, err
			}
			options = append(options, opt)
		case int:
			opt, err := handleOption(k, strconv.Itoa(v.(int)))
			if err != nil {
				return options, err
			}
			options = append(options, opt)
		case bool:
			opt, err := handleOption(k, "")
			if err != nil {
				return options, err
			}
			options = append(options, opt)
		// handles the case of a list of rules
		case interface{}:
			alt := listFieldsYAML{}
			if err := yaml.Unmarshal(data, &alt); err != nil {
				return options, fmt.Errorf("Could not unmarshal config value for key: %s", k)
			}
			if len(alt.Rule) == 0 && len(alt.JSONRule) == 0 && len(alt.Creates) == 0 {
				return options, fmt.Errorf("Unknown option: %s", k)
			}
			for _, val := range alt.Rule {
				opt, err := handleOption("rule", val)
				if err != nil {
					return options, err
				}
				options = append(options, opt)
			}
			for _, val := range alt.JSONRule {
				opt, err := handleOption("rule-json", val)
				if err != nil {
					return options, err
				}
				options = append(options, opt)
			}
			for _, val := range alt.Creates {
				opt, err := handleOption("creates", val)
				if err != nil {
					return options, err
				}
				options = append(options, opt)
			}
		default:
			return options, fmt.Errorf("Could not process config key %s, unknown type", k)
		}
	}
	return options, nil
}

type listFieldsYAML struct {
	Rule     []string `yaml:"rule"`
	JSONRule []string `yaml:"rule-json"`
	Creates  []string `yaml:"creates"`
}
