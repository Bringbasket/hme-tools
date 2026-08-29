package module

import (
	"net/http"

	"gokeep/server/internal/platform/httpx"
)

type Backend interface {
	ID() string
	RegisterRoutes(mux *http.ServeMux, auth httpx.Middleware)
	Start()
	Stop()
}
