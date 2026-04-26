#!/bin/bash

# Configuration
API_URL="http://localhost:3000/api/v1"

echo "--- 🚀 Starting Stage 1 & 2 API Tests ---"

# 1. Register a Manager
echo -e "\n1. Registering Manager (Alice)..."
curl -s -X POST "$API_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"username": "manager_alice", "email": "alice_mgr@example.com", "password": "password123", "role": "manager"}' | jq .

# 2. Login as Manager to get Token
echo -e "\n2. Logging in as Alice..."
TOKEN=$(curl -s -X POST "$API_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email": "alice_mgr@example.com", "password": "password123"}' | jq -r .token)

echo "Token: ${TOKEN:0:20}..."

# 3. Create a Team
echo -e "\n3. Creating Team (Alpha)..."
TEAM_ID=$(curl -s -X POST "$API_URL/teams/" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"teamName": "Team Alpha"}' | jq -r .teamId)
echo "Team ID: $TEAM_ID"

# 4. Register a Member
echo -e "\n4. Registering Member (Bob)..."
curl -s -X POST "$API_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"username": "member_bob", "email": "bob@example.com", "password": "password123", "role": "member"}' | jq .

# 5. Add Member to Team
echo -e "\n5. Adding Bob to Team Alpha..."
curl -s -X POST "$API_URL/teams/$TEAM_ID/members" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"userId": 2}' -v 2>&1 | grep "HTTP/1.1"

# 6. Create a Folder
echo -e "\n6. Creating a Folder (Project X)..."
FOLDER_ID=$(curl -s -X POST "$API_URL/assets/folders" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "Project X"}' | jq -r .folderId)
echo "Folder ID: $FOLDER_ID"

# 7. Create a Note in Folder
echo -e "\n7. Creating a Note in Project X..."
curl -s -X POST "$API_URL/assets/folders/$FOLDER_ID/notes" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title": "Requirements", "content": "1. Build a Go API..."}' | jq .

# 8. Test Bulk Import
echo -e "\n8. Testing Bulk User Import (CSV)..."
curl -s -X POST "$API_URL/users/import" \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@users.csv" | jq .

echo -e "\n--- ✅ Tests Completed ---"
