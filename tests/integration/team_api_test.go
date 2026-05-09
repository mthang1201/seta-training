package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/seta-training/core/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateTeam(t *testing.T) {
	ctx := context.Background()
	ts.clearTables()

	manager := ts.CreateTestUser(ctx, "manager1@test.com", "password123", domain.RoleManager)
	managerToken := ts.GenerateTestToken(manager.ID, domain.RoleManager)

	user := ts.CreateTestUser(ctx, "user1@test.com", "password123", domain.RoleMember)
	userToken := ts.GenerateTestToken(user.ID, domain.RoleMember)

	tests := []struct {
		name           string
		token          string
		payload        string
		expectedStatus int
		expectedName   string
	}{
		{
			name:           "Success - Manager creates team",
			token:          managerToken,
			payload:        `{"teamName": "Team Alpha"}`,
			expectedStatus: http.StatusCreated,
			expectedName:   "Team Alpha",
		},
		{
			name:           "Failure - Regular user forbidden",
			token:          userToken,
			payload:        `{"teamName": "Team Beta"}`,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Failure - Missing team name",
			token:          managerToken,
			payload:        `{"teamName": ""}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Failure - Invalid JSON",
			token:          managerToken,
			payload:        `{"teamName": }`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Failure - No token (Unauthorized)",
			token:          "",
			payload:        `{"teamName": "Team Gamma"}`,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := ts.PerformRequest("POST", "/api/v1/teams/", []byte(tt.payload), tt.token)
			
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusCreated {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, tt.expectedName, response["teamName"])
				assert.NotZero(t, response["teamId"])
				
				// Verify cache invalidation or publishing (implicitly checked if no errors in logs)
			}
		})
	}
}

func TestTeamMembershipConcurrency(t *testing.T) {
	ctx := context.Background()
	ts.clearTables()

	// 1. Setup Data
	manager := ts.CreateTestUser(ctx, "concurrent_manager@test.com", "password123", domain.RoleManager)
	managerToken := ts.GenerateTestToken(manager.ID, domain.RoleManager)

	// Create a team
	teamReqPayload := `{"teamName": "Concurrent Team"}`
	w := ts.PerformRequest("POST", "/api/v1/teams/", []byte(teamReqPayload), managerToken)
	require.Equal(t, http.StatusCreated, w.Code)

	var teamResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &teamResponse)
	require.NoError(t, err)
	
	teamIDRaw := teamResponse["teamId"].(float64)
	teamID := int(teamIDRaw)

	// Create 50 users to add concurrently
	userCount := 50
	var userIDs []uint
	for i := 0; i < userCount; i++ {
		email := fmt.Sprintf("member%d@test.com", i)
		user := ts.CreateTestUser(ctx, email, "password123", domain.RoleMember)
		userIDs = append(userIDs, user.ID)
	}

	// 2. Perform concurrent AddMember requests
	var wg sync.WaitGroup
	errCh := make(chan error, userCount)

	for _, uid := range userIDs {
		wg.Add(1)
		go func(userID uint) {
			defer wg.Done()
			payload := fmt.Sprintf(`{"userId": %d}`, userID)
			path := fmt.Sprintf("/api/v1/teams/%d/members", teamID)
			
			res := ts.PerformRequest("POST", path, []byte(payload), managerToken)
			if res.Code != http.StatusOK {
				errCh <- fmt.Errorf("failed to add member %d, status: %d", userID, res.Code)
			}
		}(uid)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("Concurrency error: %v", err)
	}

	// 3. Verify Final State
	team, err := ts.TeamRepo.GetByID(ctx, uint(teamID))
	require.NoError(t, err)

	// Verify all 50 members are in the team, no data races or lost updates
	assert.Equal(t, userCount, len(team.Members), "All members should be added without data loss")
}
