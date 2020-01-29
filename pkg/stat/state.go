package stat

import "github.com/BTBurke/monny/pkg/fsm"

const (
	// These represent states that the estimator can be in, starting in the reset state and progressing through testing
	// upper control limits (UCL) and the lower control limit once an alarm condition has been reached
	Reset      = fsm.State("reset")
	UCLInitial = fsm.State("ucl_initial")
	TestingUCL = fsm.State("testing_ucl")
	UCLTrip    = fsm.State("ucl_trip")
	LCLInitial = fsm.State("lcl_initial")
	TestingLCL = fsm.State("testing_lcl")
	LCLTrip    = fsm.State("lcl_trip")
)

func newMachine(initial fsm.State) (*fsm.Machine, error) {
	return fsm.NewMachine(initial, fsm.WithTransitions(
		fsm.T(Reset, UCLInitial, LCLInitial),
		fsm.T(UCLInitial, TestingUCL, Reset),
		fsm.T(TestingUCL, UCLTrip, Reset),
		fsm.T(UCLTrip, LCLInitial, Reset),
		fsm.T(LCLInitial, TestingLCL, Reset),
		fsm.T(TestingLCL, LCLTrip, Reset),
		fsm.T(LCLTrip, Reset),
	))
}
