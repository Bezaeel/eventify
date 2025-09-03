package admin_test

import (
	"bytes"
	"encoding/json"
	controllers "eventify/api/controllers/v1/admin"
	"eventify/internal/constants"
	"eventify/internal/domain"
	"eventify/tests/unit/helpers"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAssignRoleToUser(t *testing.T) {
	// Arrange
	_, mockRoleService, _, mockUserService, app, jwtProvider := setupRoleTest(t)

	token, err := helpers.GenerateValidToken(jwtProvider, constants.Permissions.AdminPermission)
	require.NoError(t, err)

	userID := uuid.New()
	roleID := uuid.New()

	request := controllers.AssignRoleRequest{
		UserID: userID.String(),
		RoleID: roleID.String(),
	}

	mockUserService.EXPECT().
		GetByID(userID).
		Return(&domain.User{ID: userID}, nil)

	mockRoleService.EXPECT().
		GetByID(roleID).
		Return(&domain.Role{Id: roleID}, nil)

	mockRoleService.EXPECT().
		AssignRoleToUser(userID, roleID).
		Return(nil)

	// Act
	body, err := json.Marshal(request)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/admin/assign-role", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)

	// Assert
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
}

// Add more test cases...
