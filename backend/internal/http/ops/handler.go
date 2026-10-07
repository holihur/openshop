package ops

import (
	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/service"
)

// Handler carries the operations-specific handlers and services on top of the
// shared base. Shared handlers are promoted from the embedded *handler.Handler.
type Handler struct {
	*handler.Handler

	Analytics *service.AnalyticsService
}
