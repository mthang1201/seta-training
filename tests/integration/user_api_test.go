package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/seta-training/core/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRegistrationAndLogin(t *testing.T) {
	ts.clearTables()

	t.Run("Success - Register user", func(t *testing.T) {
		payload := `{"email": "testregister@test.com", "password": "password123", "role": "member", "username": "testuser"}`
		w := ts.PerformRequest("POST", "/api/v1/auth/register", []byte(payload), "")

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "testregister@test.com", response["email"])
		assert.NotZero(t, response["userId"])
	})

	t.Run("Failure - Register duplicate user", func(t *testing.T) {
		payload := `{"email": "testregister@test.com", "password": "password123", "role": "member", "username": "testuser"}`
		w := ts.PerformRequest("POST", "/api/v1/auth/register", []byte(payload), "")

		// Expecting bad request or conflict depending on usecase implementation
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Success - Login user", func(t *testing.T) {
		payload := `{"email": "testregister@test.com", "password": "password123"}`
		w := ts.PerformRequest("POST", "/api/v1/auth/login", []byte(payload), "")

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotEmpty(t, response["token"])
	})

	t.Run("Failure - Login wrong password", func(t *testing.T) {
		payload := `{"email": "testregister@test.com", "password": "wrongpassword"}`
		w := ts.PerformRequest("POST", "/api/v1/auth/login", []byte(payload), "")

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestImportUsers(t *testing.T) {
	ctx := context.Background()
	ts.clearTables()

	manager := ts.CreateTestUser(ctx, "import_manager@test.com", "password123", domain.RoleManager)
	managerToken := ts.GenerateTestToken(manager.ID, domain.RoleManager)

	user := ts.CreateTestUser(ctx, "regular_user@test.com", "password123", domain.RoleMember)
	userToken := ts.GenerateTestToken(user.ID, domain.RoleMember)

	t.Run("Success - Manager bulk imports users", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Create dynamic CSV content
		csvContent := "username,email,password,role\n"
		for i := 1; i <= 10; i++ {
			csvContent += fmt.Sprintf("importuser%d,imported%d@test.com,pass123,member\n", i, i)
		}

		part, err := writer.CreateFormFile("file", "users.csv")
		require.NoError(t, err)
		part.Write([]byte(csvContent))
		writer.Close()

		req, _ := http.NewRequest("POST", "/api/v1/users/import", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+managerToken)

		w := httptest.NewRecorder()
		ts.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response domain.ImportResult
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, 10, response.Succeeded)
		assert.Equal(t, 0, response.Failed)

		// Verify in DB
		for i := 1; i <= 10; i++ {
			email := fmt.Sprintf("imported%d@test.com", i)
			u, err := ts.UserRepo.GetByEmail(ctx, email)
			require.NoError(t, err)
			assert.NotNil(t, u)
		}
	})

	t.Run("Failure - Regular user forbidden from import", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		
		csvContent := "username,email,password,role\nforbiddenuser,forbidden@test.com,pass123,member\n"
		part, _ := writer.CreateFormFile("file", "users.csv")
		part.Write([]byte(csvContent))
		writer.Close()

		req, _ := http.NewRequest("POST", "/api/v1/users/import", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+userToken)

		w := httptest.NewRecorder()
		ts.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}
