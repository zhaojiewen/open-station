package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhaojiewen/open-station/internal/application/service"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
)

// PluginHandler handles plugin management API requests
type PluginHandler struct {
	pluginService *service.PluginService
}

// NewPluginHandler creates a new plugin handler
func NewPluginHandler(pluginService *service.PluginService) *PluginHandler {
	return &PluginHandler{
		pluginService: pluginService,
	}
}

// List returns all installed plugins
func (h *PluginHandler) List(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("pageSize", "20")

	pageInt := parseInt(page, 1)
	pageSizeInt := parseInt(pageSize, 20)

	plugins, total, err := h.pluginService.List(c.Request.Context(), pageInt, pageSizeInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"plugins": plugins,
		"total":   total,
		"page":    pageInt,
		"pageSize": pageSizeInt,
	})
}

// ListAvailable returns available plugins from marketplace
func (h *PluginHandler) ListAvailable(c *gin.Context) {
	plugins, err := h.pluginService.ListAvailable(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"plugins": plugins,
	})
}

// Get returns a plugin by ID
func (h *PluginHandler) Get(c *gin.Context) {
	pluginID := c.Param("id")

	status, err := h.pluginService.Get(c.Request.Context(), pluginID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "plugin not found"})
		return
	}

	c.JSON(http.StatusOK, status)
}

// Install installs a plugin from marketplace
func (h *PluginHandler) Install(c *gin.Context) {
	pluginID := c.Param("id")

	var req struct {
		Config   map[string]interface{} `json:"config"`
		Activate bool                   `json:"activate"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		req.Config = make(map[string]interface{})
	}

	installReq := &entity.PluginInstallRequest{
		PluginID: pluginID,
		Source:   "marketplace",
		Config:   req.Config,
		Activate: req.Activate,
	}

	status, err := h.pluginService.Install(c.Request.Context(), installReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "plugin installed successfully",
		"plugin":  status,
	})
}

// Configure updates plugin configuration
func (h *PluginHandler) Configure(c *gin.Context) {
	pluginID := c.Param("id")

	var req struct {
		Config map[string]interface{} `json:"config"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.pluginService.Configure(c.Request.Context(), pluginID, req.Config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "plugin configured successfully",
	})
}

// Activate activates a plugin
func (h *PluginHandler) Activate(c *gin.Context) {
	pluginID := c.Param("id")

	if err := h.pluginService.Activate(c.Request.Context(), pluginID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "plugin activated",
		"plugin_id": pluginID,
	})
}

// Deactivate deactivates a plugin
func (h *PluginHandler) Deactivate(c *gin.Context) {
	pluginID := c.Param("id")

	if err := h.pluginService.Deactivate(c.Request.Context(), pluginID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "plugin deactivated",
		"plugin_id": pluginID,
	})
}

// Uninstall removes a plugin
func (h *PluginHandler) Uninstall(c *gin.Context) {
	pluginID := c.Param("id")

	if err := h.pluginService.Uninstall(c.Request.Context(), pluginID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "plugin uninstalled",
		"plugin_id": pluginID,
	})
}

// HealthCheck checks plugin health
func (h *PluginHandler) HealthCheck(c *gin.Context) {
	pluginID := c.Param("id")

	err := h.pluginService.HealthCheck(c.Request.Context(), pluginID)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"plugin_id": pluginID,
	})
}

// GetStats returns plugin statistics
func (h *PluginHandler) GetStats(c *gin.Context) {
	pluginID := c.Param("id")

	stats, err := h.pluginService.GetStats(c.Request.Context(), pluginID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "plugin not found"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetAllStats returns all plugin statistics
func (h *PluginHandler) GetAllStats(c *gin.Context) {
	stats, err := h.pluginService.GetAllStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stats": stats,
	})
}

// Search searches for plugins
func (h *PluginHandler) Search(c *gin.Context) {
	query := c.Query("q")

	plugins, err := h.pluginService.Search(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"plugins": plugins,
		"query":   query,
	})
}

// ByCapability returns plugins with a specific capability
func (h *PluginHandler) ByCapability(c *gin.Context) {
	capability := c.Param("capability")

	plugins, err := h.pluginService.ByCapability(c.Request.Context(), capability)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"plugins":     plugins,
		"capability":  capability,
	})
}

// GetProviders returns available providers
func (h *PluginHandler) GetProviders(c *gin.Context) {
	providers, err := h.pluginService.GetProviders(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"providers": providers,
	})
}

// parseInt helper
func parseInt(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	// Simple conversion
	var result int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result * 10 + int(c - '0')
		}
	}
	if result == 0 {
		return defaultValue
	}
	return result
}