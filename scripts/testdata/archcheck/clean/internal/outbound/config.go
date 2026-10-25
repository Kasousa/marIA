// Fixture for scripts/test-harness.sh: config.go is the one place allowed to read the environment.
package outbound

func Load() string { return os.Getenv("TABLE_NAME") }
