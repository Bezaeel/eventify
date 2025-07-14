package admin_test

import (
	"bytes"
	"encoding/json"
	controllers "eventify/internal/api/controllers/v1/admin"
	"eventify/internal/auth"
	"eventify/internal/constants"
	"eventify/internal/domain"
	service_mocks "eventify/internal/service/mocks"
	"eventify/tests/unit/helpers"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func setupRoleTest(t *testing.T) (*controllers.AdminController, *service_mocks.MockIRoleService, *service_mocks.MockIPermissionService, *service_mocks.MockIUserService, *fiber.App, auth.IJWTProvider) {
	ctrl := gomock.NewController(t)
	mockRoleService := service_mocks.NewMockIRoleService(ctrl)
	mockPermissionService := service_mocks.NewMockIPermissionService(ctrl)
	mockUserService := service_mocks.NewMockIUserService(ctrl)
	app := fiber.New()

	jwtProvider := auth.NewJWTProvider("test-secret-key", 1, "test-issuer", "test-audience")

	adminController := controllers.NewAdminController(
		app,
		mockUserService,
		mockRoleService,
		mockPermissionService,
		jwtProvider,
	)
	adminController.RegisterRoutes()

	return adminController, mockRoleService, mockPermissionService, mockUserService, app, jwtProvider
}

func TestGetAllRoles(t *testing.T) {
	// Arrange
	_, mockRoleService, _, _, app, jwtProvider := setupRoleTest(t)

	token, err := helpers.GenerateValidToken(jwtProvider, constants.Permissions.AdminPermission)
	require.NoError(t, err)

	roles := []domain.Role{
		{
			Id:          uuid.New(),
			Name:        "Admin",
			Description: "Administrator role",
			CreatedAt:   time.Now(),
			UpdatedAt:   nil,
		},
		{
			Id:          uuid.New(),
			Name:        "User",
			Description: "Regular user role",
			CreatedAt:   time.Now(),
			UpdatedAt:   nil,
		},
	}

	mockRoleService.EXPECT().
		GetAll().
		Return(roles, nil)

	// Act
	req := httptest.NewRequest("GET", "/api/v1/admin/roles", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)

	// Assert
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var responseRoles []domain.Role
	err = json.NewDecoder(resp.Body).Decode(&responseRoles)
	require.NoError(t, err)
	require.Equal(t, len(roles), len(responseRoles))
}

func TestCreateRole(t *testing.T) {
	// Arrange
	_, mockRoleService, _, _, app, jwtProvider := setupRoleTest(t)

	token, err := helpers.GenerateValidToken(jwtProvider, constants.Permissions.AdminPermission)
	require.NoError(t, err)

	newRole := controllers.CreateRoleRequest{
		Name:        "New Role",
		Description: "New role description",
	}

	mockRoleService.EXPECT().
		Create(gomock.Any()).
		Return(nil)

	// Act
	body, err := json.Marshal(newRole)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/admin/roles", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)

	// Assert
	require.NoError(t, err)
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)
}

// Add more test cases for error scenarios
func TestCreateRole_InvalidInput(t *testing.T) {
	_, _, _, _, app, jwtProvider := setupRoleTest(t)

	token, err := helpers.GenerateValidToken(jwtProvider, constants.Permissions.AdminPermission)
	require.NoError(t, err)

	invalidRole := map[string]interface{}{
		"name": 123, // Invalid type for name
	}

	body, err := json.Marshal(invalidRole)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/admin/roles", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)

	require.NoError(t, err)
	require.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}
