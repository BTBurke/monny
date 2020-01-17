package proc

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSourceFilter(t *testing.T) {
	s1 := source{
		name: pStdout,
	}
	s2 := source{
		name: pStderr,
	}
	tt := []struct {
		name   string
		in     []source
		target sourceOrSink
		expect []source
	}{
		{name: "basic filter", in: []source{s1, s2}, target: pStderr, expect: []source{s1}},
		{name: "repeated", in: []source{s2, s1, s2}, target: pStderr, expect: []source{s1}},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			out := filterSource(tc.in, tc.target)
			assert.Equal(t, tc.expect, out)
		})
	}
}

func TestSinkFilter(t *testing.T) {
	s1 := sink{
		name: mStdout,
	}
	s2 := sink{
		name: mStderr,
	}
	tt := []struct {
		name   string
		in     []sink
		target sourceOrSink
		expect []sink
	}{
		{name: "basic filter", in: []sink{s1, s2}, target: mStderr, expect: []sink{s1}},
		{name: "repeated", in: []sink{s2, s1, s2}, target: mStderr, expect: []sink{s1}},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			out := filterSink(tc.in, tc.target)
			assert.Equal(t, tc.expect, out)
		})
	}
}

func TestWithNoStderrOutput(t *testing.T) {
	l := &logProcOpt{
		sinks: []sink{sink{name: mStderr}},
	}
	f := WithNoStderrOutput()
	_ = f.apply(l)
	assert.Empty(t, l.sinks)
}

func TestWithNoStdoutOutput(t *testing.T) {
	l := &logProcOpt{
		sinks: []sink{sink{name: mStdout}},
	}
	f := WithNoStdoutOutput()
	_ = f.apply(l)
	assert.Empty(t, l.sinks)
}

func TestWithNoStdoutInput(t *testing.T) {
	l := &logProcOpt{
		sources: []source{source{name: pStdout}},
	}
	f := WithNoStdoutInput()
	_ = f.apply(l)
	assert.Empty(t, l.sources)
}

func TestWithNoStderrInput(t *testing.T) {
	l := &logProcOpt{
		sources: []source{source{name: pStderr}},
	}
	f := WithNoStderrInput()
	_ = f.apply(l)
	assert.Empty(t, l.sources)
}

func TestWithNoOutput(t *testing.T) {
	l := &logProcOpt{
		sinks: []sink{sink{name: mStdout}, sink{name: mStderr}},
	}
	f := WithNoOutput()
	_ = f.apply(l)
	assert.Empty(t, l.sinks)
}

func TestWithCommand(t *testing.T) {
	tt := []struct {
		name    string
		cmd     *exec.Cmd
		sources []sourceOrSink
		sinks   []sourceOrSink
	}{
		{name: "no pipe", cmd: exec.Command("test"), sources: []sourceOrSink{pStdout, pStderr}, sinks: []sourceOrSink{mStdout, mStderr}},
		{name: "pipe", cmd: exec.Command(""), sources: []sourceOrSink{mStdin}, sinks: []sourceOrSink{mStdout}},
	}
	extractSources := func(sources []source) (out []sourceOrSink) {
		for _, s := range sources {
			out = append(out, s.name)
		}
		return
	}
	extractSinks := func(sinks []sink) (out []sourceOrSink) {
		for _, s := range sinks {
			out = append(out, s.name)
		}
		return
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			l := &logProcOpt{}
			f := WithCommand(tc.cmd)
			_ = f.apply(l)
			assert.Equal(t, tc.sources, extractSources(l.sources))
			assert.Equal(t, tc.sinks, extractSinks(l.sinks))
		})
	}
}
