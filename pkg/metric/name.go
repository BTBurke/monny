package metric

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/go-logfmt/logfmt"
)

type metadata map[string]string

// Name is an identifier for the metric.  By convention, the name typically ends in the metric type, such as
// requests_count, disk_latency_gauge, etc.  Optional metadata is added to the name to group similar metric types
// to help find the source of the error.  Names are marshalled to a string using a modified logfmt, e.g.
// requests_count[host=pod1 loc=us-west1]
type Name struct {
	name string
	md   metadata
}

// String marshals the name to a string representation, such as requests_count[host=pod1 loc=us-west1]
func (n Name) String() string {
	md, err := MarshalText(n.md)
	if err != nil {
		md = []byte{}
	}
	return n.name + string(md)
}

// NewName returns a new name with the associated metadata
func NewName(name string, md map[string]string) Name {
	return Name{name: name, md: md}
}

// AddAnnotation adds additional annotations
func (n Name) AddAnnotation(ann ...string) {
	for _, a := range ann {
		n.md[a] = ""
	}
}

// AddMetadata adds additional metadata upserted into the metadata map.
func (n Name) AddMetadata(md map[string]string) {
	for k, v := range md {
		n.md[k] = v
	}
}

// MarshalText will return the metadata encoded as a modified logfmt representation.  Metadata opens with a [
// then is followed by (key, value) pairs k=v in sorted key order, the finally by annotations starting with @ in
// sorted order.  Close with a ].  Example: [host=pod1 loc=us-west-1 @mean @summary]
func MarshalText(m metadata) ([]byte, error) {
	if m == nil {
		return []byte{}, nil
	}
	keys := make([]string, 0, len(m))
	ann := make([]string, 0, len(m))
	for k, v := range m {
		switch v {
		case "":
			ann = append(ann, fmt.Sprintf("@%s", k))
		default:
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	sort.Strings(ann)

	var b bytes.Buffer
	b.Write([]byte("["))
	e := logfmt.NewEncoder(&b)
	for _, k := range keys {
		if err := e.EncodeKeyval(k, m[k]); err != nil {
			return nil, fmt.Errorf("failed to encode %s=%s: %v", k, m[k], err)
		}
	}
	if len(keys) > 0 && len(ann) > 0 {
		b.Write([]byte(" "))
	}
	if len(ann) > 0 {
		b.Write([]byte(strings.Join(ann, " ")))
	}
	b.Write([]byte("]"))
	return b.Bytes(), nil
}

func NewNameFrom(n Name) Name {
	copiedMD := make(map[string]string)
	for k, v := range n.md {
		copiedMD[k] = v
	}
	return NewName(n.name, copiedMD)
}
