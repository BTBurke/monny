package proc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueue(t *testing.T) {
	tt := []struct {
		Name string
		In   []string
		Exp  []string
	}{
		{"empty queue", []string{}, []string{}},
		{"non filled queue", []string{"1"}, []string{"1"}},
		{"filled queue", []string{"1", "2"}, []string{"1", "2"}},
		{"overfilled queue", []string{"1", "2", "3"}, []string{"2", "3"}},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			q := NewQueue(2)
			for _, s := range tc.In {
				q.Add(s)
			}
			scopy := q.Copy()
			assert.Equal(t, q.q, tc.Exp)
			assert.Equal(t, scopy, tc.Exp)
			q.Clear()
			assert.Equal(t, q.q, []string{})
		})
	}
}
