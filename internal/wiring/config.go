package wiring

// Config defines the app config
type Config struct {
	DecisionServiceEndpoint string
	EnableLeaderElection    bool
	LogLevel                string
	MetricsAddr             string
}
