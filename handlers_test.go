package monny

import (
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/BTBurke/monny/proto"
	"github.com/stretchr/testify/mock"
)

type mockRep struct {
	mock.Mock
}

func (m *mockRep) Send(c *Command, reason proto.ReportReason) {
	m.Called()
	return
}

func (m *mockRep) Wait() error {
	return nil
}

func TestSuccessHandler(t *testing.T) {
	c, err := New([]string{"test"}, ID("test"))
	if err != nil {
		t.Fatalf("unexpected error creating command: %s", err)
	}
	mocks := &mockRep{}
	c.report = mocks
	mocks.On("Send").Return()

	cmd := exec.Command("sleep", "1")
	if err := cmd.Run(); err != nil {
		t.Fatalf("unexpected error running command: %s", err)
	}
	h := handler{}
	errHandle := h.Finished(c, cmd)

	assert.Nil(t, errHandle)
	assert.Equal(t, proto.Success, c.ReportReason)
	assert.NotZero(t, c.Duration)
	assert.True(t, c.Success)
	//mocks.AssertExpectations(t)
}

func TestFailureHandler(t *testing.T) {
	c, errs := New([]string{"test"}, ID("test"))
	if len(errs) != 0 {
		t.Fatalf("unexpected error creating command: %s", errs)
	}
	mockR := new(mockRep)
	c.report = mockR
	mockR.On("Send").Return(nil)

	f, err := ioutil.TempFile("", "xrtest")
	if err != nil {
		t.Fatalf("unexpected error creating temp cmd: %s", err)
	}
	defer os.Remove(f.Name())
	if _, err := f.Write([]byte("#!/bin/bash\nexit 1")); err != nil {
		t.Fatalf("unexpected error writing temp cmd: %s", err)
	}
	if err := f.Chmod(os.ModePerm); err != nil {
		t.Fatalf("unexpected error setting permissions: %s", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("unexpected error closing file: %s", err)
	}
	cmd := exec.Command(f.Name())
	cmd.Run()

	h := handler{}
	errHandle := h.Finished(c, cmd)

	assert.Nil(t, errHandle)
	assert.Equal(t, proto.Failure, c.ReportReason)
	assert.NotZero(t, c.Duration)
	assert.False(t, c.Success)
}

func TestSignalHandler(t *testing.T) {
	c, errs := New([]string{"test"}, ID("test"))
	if len(errs) != 0 {
		t.Fatalf("unexpected error creating command: %s", errs)
	}
	mocks := new(mockRep)
	c.report = mocks
	mocks.On("Send").Return()

	f, err := ioutil.TempFile("", "xrtest")
	if err != nil {
		t.Fatalf("unexpected error creating temp cmd: %s", err)
	}
	defer os.Remove(f.Name())
	if _, err := f.Write([]byte("#!/bin/bash\nsleep 10")); err != nil {
		t.Fatalf("unexpected error writing temp cmd: %s", err)
	}
	if err := f.Chmod(os.ModePerm); err != nil {
		t.Fatalf("unexpected error setting permissions: %s", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("unexpected error closing file: %s", err)
	}
	cmd := exec.Command(f.Name())
	if err := cmd.Start(); err != nil {
		t.Fatalf("unexpected error starting process: %s", err)
	}

	h := handler{}
	errHandle := h.Signal(c, cmd, os.Kill)

	assert.Nil(t, errHandle)
	assert.Equal(t, proto.Killed, c.ReportReason)
	assert.Equal(t, proto.Signal, c.KillReason)
	assert.NotZero(t, c.Duration)
	assert.False(t, c.Success)
}

func TestKillMemoryHandler(t *testing.T) {
	c, errs := New([]string{"test"}, ID("test"))
	if len(errs) != 0 {
		t.Fatalf("unexpected error creating command: %s", errs)
	}
	mocks := new(mockRep)
	c.report = mocks
	mocks.On("Send").Return()

	f, err := ioutil.TempFile("", "xrtest")
	if err != nil {
		t.Fatalf("unexpected error creating temp cmd: %s", err)
	}
	defer os.Remove(f.Name())
	if _, err := f.Write([]byte("#!/bin/bash\nsleep 10")); err != nil {
		t.Fatalf("unexpected error writing temp cmd: %s", err)
	}
	if err := f.Chmod(os.ModePerm); err != nil {
		t.Fatalf("unexpected error setting permissions: %s", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("unexpected error closing file: %s", err)
	}
	cmd := exec.Command(f.Name())
	if err := cmd.Start(); err != nil {
		t.Fatalf("unexpected error starting process: %s", err)
	}

	h := handler{}
	errHandle := h.KillOnHighMemory(c, cmd)

	assert.Nil(t, errHandle)
	assert.Equal(t, proto.Killed, c.ReportReason)
	assert.Equal(t, proto.Memory, c.KillReason)
	assert.NotZero(t, c.Duration)
	assert.False(t, c.Success)
}

func TestKillTimeoutHandler(t *testing.T) {
	c, errs := New([]string{"test"}, ID("test"))
	if len(errs) != 0 {
		t.Fatalf("unexpected error creating command: %s", errs)
	}
	mocks := new(mockRep)
	c.report = mocks
	mocks.On("Send").Return()

	f, err := ioutil.TempFile("", "xrtest")
	if err != nil {
		t.Fatalf("unexpected error creating temp cmd: %s", err)
	}
	defer os.Remove(f.Name())
	if _, err := f.Write([]byte("#!/bin/bash\nsleep 10")); err != nil {
		t.Fatalf("unexpected error writing temp cmd: %s", err)
	}
	if err := f.Chmod(os.ModePerm); err != nil {
		t.Fatalf("unexpected error setting permissions: %s", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("unexpected error closing file: %s", err)
	}
	cmd := exec.Command(f.Name())
	if err := cmd.Start(); err != nil {
		t.Fatalf("unexpected error starting process: %s", err)
	}

	h := handler{}
	errHandle := h.Timeout(c, cmd)

	assert.Nil(t, errHandle)
	assert.Equal(t, proto.Killed, c.ReportReason)
	assert.Equal(t, proto.Timeout, c.KillReason)
	assert.NotZero(t, c.Duration)
	assert.False(t, c.Success)
}

func TestCheckMemoryHandler(t *testing.T) {
	c, errs := New([]string{"test"}, ID("test"), MemoryWarn("1K"))
	if len(errs) != 0 {
		t.Fatalf("unexpected error creating command: %s", errs)
	}
	mocks := new(mockRep)
	c.report = mocks
	mocks.On("Send").Return()

	f, err := ioutil.TempFile("", "xrtest")
	if err != nil {
		t.Fatalf("unexpected error creating temp cmd: %s", err)
	}
	defer os.Remove(f.Name())
	if _, err := f.Write([]byte("#!/bin/bash\nsleep 2")); err != nil {
		t.Fatalf("unexpected error writing temp cmd: %s", err)
	}
	if err := f.Chmod(os.ModePerm); err != nil {
		t.Fatalf("unexpected error setting permissions: %s", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("unexpected error closing file: %s", err)
	}
	cmd := exec.Command(f.Name())
	if err := cmd.Start(); err != nil {
		t.Fatalf("unexpected error starting process: %s", err)
	}

	h := handler{}
	errHandle := h.CheckMemory(c, cmd)

	assert.Nil(t, errHandle)
	assert.Equal(t, proto.MemoryWarning, c.ReportReason)
	assert.True(t, c.memWarnSent)
	assert.NotZero(t, c.MaxMemory)
}

func TestTimeWarnHandler(t *testing.T) {
	c, err := New([]string{"test"}, ID("test"))
	if err != nil {
		t.Fatalf("unexpected error creating command: %s", err)
	}
	mocks := &mockRep{}
	c.report = mocks
	mocks.On("Send").Return()

	h := handler{}
	errHandle := h.TimeWarning(c)

	assert.Nil(t, errHandle)
	assert.Equal(t, proto.TimeWarning, c.ReportReason)
	assert.True(t, c.timeWarnSent)
}
