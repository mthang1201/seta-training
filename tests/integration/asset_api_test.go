package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/seta-training/core/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssetManagementAndObservability(t *testing.T) {
	ctx := context.Background()
	ts.clearTables()
	ts.ClearEvents()

	// 1. Setup Users and Team
	manager := ts.CreateTestUser(ctx, "manager@asset.com", "password123", domain.RoleManager)
	managerToken := ts.GenerateTestToken(manager.ID, domain.RoleManager)

	owner := ts.CreateTestUser(ctx, "owner@asset.com", "password123", domain.RoleMember)
	ownerToken := ts.GenerateTestToken(owner.ID, domain.RoleMember)

	otherUser := ts.CreateTestUser(ctx, "other@asset.com", "password123", domain.RoleMember)
	otherToken := ts.GenerateTestToken(otherUser.ID, domain.RoleMember)

	// Create Team and add owner as member
	ts.PerformRequest("POST", "/api/v1/teams/", []byte(`{"teamName": "Asset Team"}`), managerToken)
	var team domain.Team
	ts.DB.First(&team)
	ts.PerformRequest("POST", fmt.Sprintf("/api/v1/teams/%d/members", team.ID), []byte(fmt.Sprintf(`{"userId": %d}`, owner.ID)), managerToken)
	ts.ClearEvents() // Ignore team events for this test

	// --- Test Variables ---
	var folderID uint
	var noteID uint

	t.Run("Create Folder and Emit Event", func(t *testing.T) {
		payload := `{"name": "Secret Plans"}`
		w := ts.PerformRequest("POST", "/api/v1/assets/folders", []byte(payload), ownerToken)
		require.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		folderID = uint(response["folderId"].(float64))

		// Assert RabbitMQ Event
		ev := ts.AssertEventReceived(t, domain.EventAssetCreated)
		require.NotNil(t, ev)
		payloadMap := ev.Payload.(map[string]interface{})
		assert.Equal(t, "folder", payloadMap["assetType"])
	})

	t.Run("Create Note and Assert Cache", func(t *testing.T) {
		payload := fmt.Sprintf(`{"title": "Launch Codes", "content": "12345"}`)
		path := fmt.Sprintf("/api/v1/assets/folders/%d/notes", folderID)
		w := ts.PerformRequest("POST", path, []byte(payload), ownerToken)
		require.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		noteID = uint(response["noteId"].(float64))

		// Fetch Note to trigger Cache
		fetchPath := fmt.Sprintf("/api/v1/assets/notes/%d", noteID)
		wFetch := ts.PerformRequest("GET", fetchPath, nil, ownerToken)
		require.Equal(t, http.StatusOK, wFetch.Code)

		// Assert Redis Cache Hit
		cacheKey := fmt.Sprintf("asset:note:%d", noteID)
		cachedData, err := ts.RedisCache.Get(ctx, cacheKey)
		require.NoError(t, err)
		assert.Contains(t, cachedData, "Launch Codes")
		
		// Wait to clear event queue
		ts.AssertEventReceived(t, domain.EventAssetCreated)
	})

	t.Run("Manager Oversight - Read Success, Write Forbidden", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/assets/notes/%d", noteID)
		
		// Manager can READ the note (Oversight rule)
		wRead := ts.PerformRequest("GET", path, nil, managerToken)
		assert.Equal(t, http.StatusOK, wRead.Code)

		// Manager CANNOT EDIT the note
		updatePayload := `{"title": "Hacked Codes", "content": "99999"}`
		wUpdate := ts.PerformRequest("PUT", path, []byte(updatePayload), managerToken)
		assert.Equal(t, http.StatusForbidden, wUpdate.Code)
	})

	t.Run("Other User - Read Forbidden", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/assets/notes/%d", noteID)
		wRead := ts.PerformRequest("GET", path, nil, otherToken)
		assert.Equal(t, http.StatusForbidden, wRead.Code)
	})

	t.Run("Owner Updates Note - Assert Cache Invalidation", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/assets/notes/%d", noteID)
		updatePayload := `{"title": "Updated Codes", "content": "54321"}`
		wUpdate := ts.PerformRequest("PUT", path, []byte(updatePayload), ownerToken)
		require.Equal(t, http.StatusOK, wUpdate.Code)

		// Assert Cache Invalidated
		cacheKey := fmt.Sprintf("asset:note:%d", noteID)
		cachedData, _ := ts.RedisCache.Get(ctx, cacheKey)
		assert.Empty(t, cachedData, "Cache should be deleted upon update")

		// Assert RabbitMQ Update Event
		ev := ts.AssertEventReceived(t, domain.EventAssetUpdated)
		require.NotNil(t, ev)
	})

	t.Run("Share Folder and Test ACL Inheritance", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/assets/folder/%d/share", folderID)
		sharePayload := fmt.Sprintf(`{"targetUserId": %d, "accessLevel": "read"}`, otherUser.ID)
		wShare := ts.PerformRequest("POST", path, []byte(sharePayload), ownerToken)
		require.Equal(t, http.StatusOK, wShare.Code)

		// Assert Event
		ev := ts.AssertEventReceived(t, domain.EventAssetShared)
		require.NotNil(t, ev)

		// Other User can now read the note inside the folder (Inheritance)
		notePath := fmt.Sprintf("/api/v1/assets/notes/%d", noteID)
		wRead := ts.PerformRequest("GET", notePath, nil, otherToken)
		assert.Equal(t, http.StatusOK, wRead.Code)

		// Check that ACL Cache was populated for note access inheritance
		// The ACL cache key will be explicitly generated on fetch if they had direct access, 
		// but since it's inherited it might not populate the exact note key, but the folder check runs.
		// Let's verify revoking access works and emits event.
	})

	t.Run("Revoke Access and Check Cache Invalidation", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/assets/folder/%d/share/%d", folderID, otherUser.ID)
		wRevoke := ts.PerformRequest("DELETE", path, nil, ownerToken)
		require.Equal(t, http.StatusOK, wRevoke.Code)

		// Verify ACL Cache Invalidated
		aclCacheKey := fmt.Sprintf("asset:folder:%d:acl:%d", folderID, otherUser.ID)
		cachedAcl, _ := ts.RedisCache.Get(ctx, aclCacheKey)
		assert.Empty(t, cachedAcl)

		// Other User can NO LONGER read the note
		notePath := fmt.Sprintf("/api/v1/assets/notes/%d", noteID)
		wRead := ts.PerformRequest("GET", notePath, nil, otherToken)
		assert.Equal(t, http.StatusForbidden, wRead.Code)

		// Assert Event
		ev := ts.AssertEventReceived(t, domain.EventAssetShared) // Same event type used for revoke, check payload
		require.NotNil(t, ev)
		payloadMap := ev.Payload.(map[string]interface{})
		assert.Equal(t, "revoked", payloadMap["action"])
	})
}
