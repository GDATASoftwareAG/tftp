package metrics

const (
	defaultMetricsRoute = "/metrics"
)

type Config struct {
	Enabled bool
	Route   string
	Port    uint16
}

func (c Config) GetMetricsRoute() string {
	if len(c.Route) == 0 {
		return defaultMetricsRoute
	}
	return c.Route
}
