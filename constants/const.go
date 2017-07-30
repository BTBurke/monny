//go:generate stringer -type=KillReason,ReportReason -output const_string.go
package constants

type KillReason int

const (
	Timeout KillReason = iota + 1
	Memory
	Signal
)

type ReportReason int

const (
	Success ReportReason = iota + 1
	Failure
	Alert
	AlertRate
	MemoryWarning
	TimeWarning
	FileNotCreated
	Killed
	Start
)
