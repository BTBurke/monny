package stat

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestK(t *testing.T) {
	tt := []struct {
		e float64
		k float64
	}{
		{e: .05, k: 5.2684},
		{e: .01, k: 5.6921},
		{e: .001, k: 6.2983},
	}
	for _, tc := range tt {
		k := K{tc.e}
		kk, err := k.Calculate()
		assert.NoError(t, err)
		assert.InDelta(t, tc.k, kk, .02)
	}
}
