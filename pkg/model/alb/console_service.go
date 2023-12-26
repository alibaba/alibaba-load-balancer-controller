package alb

type ConsoleServiceStack struct {
	ClusterID     string
	ServerGroupID string

	Namespace string
	Name      string

	TrafficPolicy                   string
	ContainsPotentialReadyEndpoints bool
	Backends                        []BackendItem
}
