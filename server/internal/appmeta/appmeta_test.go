package appmeta

import (
	"slices"
	"testing"
)

func TestSupportedServices(t *testing.T) {
	want := []ServiceName{
		ServiceAPI,
		ServiceScheduler,
		ServiceAggregator,
	}

	if got := SupportedServices(); !slices.Equal(got, want) {
		t.Fatalf("SupportedServices() = %v, want %v", got, want)
	}
}

func TestSupportedServicesReturnsCopy(t *testing.T) {
	services := SupportedServices()
	services[0] = "unexpected"

	if got := SupportedServices()[0]; got != ServiceAPI {
		t.Fatalf("SupportedServices() shared internal slice, got %q", got)
	}
}

func TestIsSupportedService(t *testing.T) {
	if !IsSupportedService(ServiceScheduler) {
		t.Fatal("expected scheduler to be supported")
	}

	if IsSupportedService("worker") {
		t.Fatal("did not expect worker to be supported in round 1 bootstrap")
	}
}
