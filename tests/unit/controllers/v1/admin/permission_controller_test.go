package admin_test

import (
	"encoding/json"
	"eventify/internal/constants"
	"eventify/internal/domain"
	"eventify/tests/unit/helpers"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestGetAllPermissions(t *testing.T) {
	// Arrange
	_, _, mockPermService, _, app, jwtProvider := setupRoleTest(t)

	token, err := helpers.GenerateValidToken(jwtProvider, constants.Permissions.AdminPermission)
	require.NoError(t, err)

	permissions := []domain.Permission{
		{
			Id:          uuid.New(),
			Name:        "read:users",
			Description: "Can read users",
		},
		{
			Id:          uuid.New(),
			Name:        "write:users",
			Description: "Can write users",
		},
	}

	mockPermService.EXPECT().
		GetAll().
		Return(permissions, nil)

	// Act
	req := httptest.NewRequest("GET", "/api/v1/admin/permissions", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)

	// Assert
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var responsePerms []domain.Permission
	err = json.NewDecoder(resp.Body).Decode(&responsePerms)
	require.NoError(t, err)
	require.Equal(t, len(permissions), len(responsePerms))
}



// Add more test cases...
