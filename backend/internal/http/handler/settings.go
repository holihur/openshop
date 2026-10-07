package handler

import (
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/response"
)

// SettingView is one runtime setting for the ops console.
type SettingView struct {
	Key         string `json:"key"`
	Group       string `json:"group"`
	Type        string `json:"type"`
	Value       string `json:"value"`
	Default     string `json:"default"`
	Description string `json:"description"`
	Min         int    `json:"min,omitempty"`
	Max         int    `json:"max,omitempty"`
}

func toSettingViews(items []domain.Setting) []SettingView {
	out := make([]SettingView, 0, len(items))
	for _, s := range items {
		out = append(out, SettingView{
			Key: s.Key, Group: s.Group, Type: string(s.Type),
			Value: s.Value, Default: s.Default, Description: s.Description, Min: s.Min, Max: s.Max,
		})
	}
	return out
}

// ListSettings returns every runtime setting with its current and default value.
func (h *Handler) ListSettings(c *gin.Context) {
	response.OK(c, toSettingViews(h.Settings.List(c.Request.Context())))
}

// UpdateSettings applies a batch of runtime setting overrides and returns the
// resolved settings. Changes take effect without a restart.
func (h *Handler) UpdateSettings(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, WrapBind(err))
		return
	}
	if err := h.Settings.Update(c.Request.Context(), req); err != nil {
		response.Fail(c, err)
		return
	}
	keys := make([]string, 0, len(req))
	for k := range req {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h.RecordAudit(c, "settings.update", "settings", "", map[string]string{"keys": strings.Join(keys, ",")})
	response.OK(c, toSettingViews(h.Settings.List(c.Request.Context())))
}
