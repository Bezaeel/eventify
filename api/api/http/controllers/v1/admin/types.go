package controllers

// Request types
// @Description Role creation request
type CreateRoleRequest struct {
	// Name of the role
	Name string `json:"name" validate:"required" example:"admin"`
	// Description of the role
	Description string `json:"description" example:"Administrator role with full access"`
}

// @Description Role assignment request
type AssignRoleRequest struct {
	// ID of the user to assign the role to
	UserID string `json:"user_id" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	// ID of the role to assign
	RoleID string `json:"role_id" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440001"`
}

// @Description Permission assignment request
type AssignPermissionRequest struct {
	// ID of the role to assign the permission to
	RoleID string `json:"role_id" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	// ID of the permission to assign
	PermissionID string `json:"permission_id" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440001"`
}

// Response types
// @Description Generic success response
type SuccessResponse struct {
	// Success message
	Message string `json:"message" example:"Operation completed successfully"`
}
