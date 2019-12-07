package monny

import "strings"

func envToKeyValue(env map[string]string) []string {
	var out []string
	for k, v := range env {
		out = append(out, strings.ToUpper(k)+"="+v)
	}
	return out
}
