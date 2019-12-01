//go:generate stringer -type=KillReason,ReportReason -output enum_string.go
package proto

type ReportReason int32

const (
	_ ReportReason = iota
	Success
	Failure
	Alert
	AlertRate
	MemoryWarning
	TimeWarning
	FileNotCreated
	Killed
	Start
)

type KillReason int32

const (
	_ KillReason = iota
	Timeout
	Memory
	Signal
)
