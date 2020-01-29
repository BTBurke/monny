package metric

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNameMarshal(t *testing.T) {
	tt := []struct {
		name string
		n    string
		md   map[string]string
		exp  string
	}{
		{name: "no metadata", n: "test_counter", exp: "test_counter"},
		{name: "metadata", n: "test_counter", md: map[string]string{"host": "pod", "loc": "us-west-1"}, exp: "test_counter[host=pod loc=us-west-1]"},
		{name: "metadata spaces", n: "test_counter", md: map[string]string{"loc": "us west 1", "host": "pod"}, exp: "test_counter[host=pod loc=\"us west 1\"]"},
		{name: "metadata with annotations", n: "test_counter", md: map[string]string{"loc": "us-west-1", "host": "pod", "mean": ""}, exp: "test_counter[host=pod loc=us-west-1 @mean]"},
		{name: "annotations only", n: "test_counter", md: map[string]string{"mean": "", "sampled": ""}, exp: "test_counter[@mean @sampled]"},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			n := NewName(tc.n, tc.md)
			assert.Equal(t, tc.exp, n.String())
		})
	}
}

func TestAddMetadata(t *testing.T) {
	tt := []struct {
		name string
		add  map[string]string
		exp  map[string]string
	}{
		{name: "no replacement", add: map[string]string{"c": "d", "e": "f"}, exp: map[string]string{"a": "b", "c": "d", "e": "f"}},
		{name: "replacement", add: map[string]string{"a": "d", "e": "f"}, exp: map[string]string{"a": "d", "e": "f"}},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ini := map[string]string{"a": "b"}
			n := NewName("test", ini)
			n.AddMetadata(tc.add)
			assert.Equal(t, metadata(tc.exp), n.md)
		})
	}
}

func TestAddAnnotation(t *testing.T) {
	tt := []struct {
		name string
		add  []string
		exp  map[string]string
	}{
		{name: "add", add: []string{"c", "e"}, exp: map[string]string{"a": "b", "c": "", "e": ""}},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ini := map[string]string{"a": "b"}
			n := NewName("test", ini)
			n.AddAnnotation(tc.add...)
			assert.Equal(t, metadata(tc.exp), n.md)
		})
	}
}
