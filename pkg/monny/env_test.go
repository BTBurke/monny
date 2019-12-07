package monny

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEnvFormatter(t *testing.T) {
	tt := []struct{
		Name string
		In map[string]string
		Out []string
	}{
		{Name: "basic", In: map[string]string{"k1": "v1", "k2": "v2"}, Out: []string{"K1=v1", "K2=v2"}},
	}
	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			out := envToKeyValue(tc.In)
			assert.ElementsMatch(t, out, tc.Out)
		})
	}
}
