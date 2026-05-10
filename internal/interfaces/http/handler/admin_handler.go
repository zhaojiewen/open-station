package handler

// AdminHandler handles admin-related requests (placeholder for future implementation)
type AdminHandler struct {
	userRepo   interface{}
	tenantRepo interface{}
	modelRepo  interface{}
}

func NewAdminHandler() *AdminHandler {
	return &AdminHandler{}
}