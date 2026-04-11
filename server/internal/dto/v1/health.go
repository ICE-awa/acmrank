package v1

type DependencyHealthResponse struct {
	Name       string `json:"name"`
	Configured bool   `json:"configured"`
	Reachable  bool   `json:"reachable"`
	Message    string `json:"message"`
}

type HealthResponse struct {
	Service      string                     `json:"service"`
	Version      string                     `json:"version"`
	Status       string                     `json:"status"`
	StartedAt    string                     `json:"started_at"`
	CheckedAt    string                     `json:"checked_at"`
	Dependencies []DependencyHealthResponse `json:"dependencies"`
}
