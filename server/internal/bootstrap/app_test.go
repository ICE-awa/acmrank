package bootstrap

import (
	"net/http"
	"testing"
	"time"

	"github.com/ICE-awa/acmrank/server/internal/appmeta"
	"github.com/ICE-awa/acmrank/server/internal/config"
)

func TestNewHTTPServerAppliesTimeouts(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Service:           appmeta.ServiceAPI,
		HTTPAddr:          ":8080",
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       12 * time.Second,
		WriteTimeout:      18 * time.Second,
		IdleTimeout:       70 * time.Second,
	}

	server := newHTTPServer(cfg, http.NewServeMux())

	if server.ReadHeaderTimeout != 5*time.Second {
		t.Fatalf("ReadHeaderTimeout = %v, want %v", server.ReadHeaderTimeout, 5*time.Second)
	}

	if server.ReadTimeout != 12*time.Second {
		t.Fatalf("ReadTimeout = %v, want %v", server.ReadTimeout, 12*time.Second)
	}

	if server.WriteTimeout != 18*time.Second {
		t.Fatalf("WriteTimeout = %v, want %v", server.WriteTimeout, 18*time.Second)
	}

	if server.IdleTimeout != 70*time.Second {
		t.Fatalf("IdleTimeout = %v, want %v", server.IdleTimeout, 70*time.Second)
	}
}
