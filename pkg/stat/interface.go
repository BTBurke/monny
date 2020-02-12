package stat

// Test defines methods available on tests which may include several statistical sub estimators using various
// techniques to detect changes.  Default estimators use both EWMA and Shewart in parallel.  Alarm conditions are
// true if any statistic is in an alarmed condition.  Manually transitioning the test to a different state will
// attempt to force every sub test to the same desired state.
// type Test interface {
// Name() string
// Record(obs float64) error
// State() []fsm.State
// Transition(s fsm.State, reset bool) error
// HasAlarmed() bool
// Metric() map[string]float64
// Done()
// }
//
//Statistic defines methods available on any type of test statistic (e.g. EWMA, Shewart)
// type Statistic interface {
// Name() string
// Record(s float64) error
// State() fsm.State
// Transition(s fsm.State, reset bool) error
// HasAlarmed() bool
// Value() float64
// Limit() float64
// Done()
// }
