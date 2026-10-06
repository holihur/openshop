package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/http/response"
	"github.com/holihur/openshop/internal/version"
)

// Healthz is a liveness probe: it only reports that the process is running.
// It must stay dependency-free so a slow database never causes a restart loop.
func (h *Handler) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Readyz is a readiness probe: it verifies every downstream dependency so the
// load balancer stops routing to an unhealthy replica.
func (h *Handler) Readyz(c *gin.Context) {
	ctx := c.Request.Context()
	results := make(map[string]string, len(h.Checks))
	healthy := true
	for _, check := range h.Checks {
		if err := check.Check(ctx); err != nil {
			results[check.Name] = err.Error()
			healthy = false
			continue
		}
		results[check.Name] = "ok"
	}
	if !healthy {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "checks": results})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "checks": results})
}

// Version reports the build metadata injected at release time.
func (h *Handler) Version(c *gin.Context) {
	response.OK(c, gin.H{
		"name":    "openshop",
		"version": version.Version,
		"commit":  version.Commit,
		"date":    version.Date,
	})
}
