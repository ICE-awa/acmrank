package appmeta

type ServiceName string

const (
	ServiceAPI        ServiceName = "api"
	ServiceScheduler  ServiceName = "scheduler"
	ServiceAggregator ServiceName = "aggregator"
)

var supportedServices = []ServiceName{
	ServiceAPI,
	ServiceScheduler,
	ServiceAggregator,
}

func SupportedServices() []ServiceName {
	services := make([]ServiceName, len(supportedServices))
	copy(services, supportedServices)

	return services
}

func IsSupportedService(name ServiceName) bool {
	for _, candidate := range supportedServices {
		if candidate == name {
			return true
		}
	}

	return false
}
