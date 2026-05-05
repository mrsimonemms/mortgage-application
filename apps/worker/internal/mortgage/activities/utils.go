package activities

import (
	"math/rand"
	"time"
)

// MinActivityDelay and MaxActivityDelay define the simulated delay range applied to
// every activity. They are package-level vars (not consts) so that test code can
// zero them out to keep the test suite fast.
var (
	MinActivityDelay = 500 * time.Millisecond
	MaxActivityDelay = 3 * time.Second
)

// DisableActivityDelaysForTests sets both delay bounds to zero so that activities
// return immediately during test execution. Call this from an init() function in
// any test package that executes real activities (including workflow tests).
func DisableActivityDelaysForTests() {
	MinActivityDelay = 0
	MaxActivityDelay = 0
}

// randomActivityDelay returns a pseudo-random duration in [MinActivityDelay, MaxActivityDelay).
// Returns MinActivityDelay when the range is zero or inverted, which avoids a panic
// from rand.Int63n(0) and keeps the function safe if the bounds are narrowed for tests.
// Must only be called from activity implementations, never from workflow code.
func randomActivityDelay() time.Duration {
	if MaxActivityDelay <= MinActivityDelay {
		return MinActivityDelay
	}
	delta := MaxActivityDelay - MinActivityDelay
	return MinActivityDelay + time.Duration(rand.Int63n(int64(delta)))
}
