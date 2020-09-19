package wiring

// Config defines the app config
type Config struct {
	EnableLeaderElection bool
	LogLevel             string
	MetricsAddr          string
}
