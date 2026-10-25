// Fixture for scripts/test-harness.sh: clock.go is the one place allowed to call time.Now().
package outbound

func (realClock) Now() time.Time { return time.Now() }
