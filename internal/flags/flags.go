package flags

import (
	"os"

	"gopkg.in/alecthomas/kingpin.v2"

	"hpa-tuner/internal/wiring"
)

// Flags parses the commandline flags or environmental variables
func Flags(name, gitCommit, version string, cfg *wiring.Config) *kingpin.Application {

	app := kingpin.New(name, "An hpa-tuner kubernetes using a custom controller")
	app.Flag("enable-leader-election", "Enable leader election, this will ensure there is only one active controller manager").Short('e').Envar("ENABLE_LEADER_ELECTION").BoolVar(&cfg.EnableLeaderElection)
	app.Flag("loglevel", `log level: "debug", "info", "warn", "error", "dpanic", "panic", and "fatal".`).Short('l').Envar("LOG_LEVEL").Default("info").EnumVar(&cfg.LogLevel, "debug", "info", "warn", "error", "dpanic", "panic", "fatal")
	app.Flag("tmetrics-addr", "The address the metric endpoint binds to").Short('m').Envar("METRICS_ADDR").StringVar(&cfg.MetricsAddr)

	kingpin.MustParse(app.Parse(os.Args[1:]))

	return app
}
