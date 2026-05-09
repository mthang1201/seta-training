package domain

import (
	"context"
	"time"
)

// Event Types
const (
	EventTeamCreated   = "TEAM_CREATED"
	EventMemberAdded   = "MEMBER_ADDED"
	EventMemberRemoved = "MEMBER_REMOVED"
	EventAssetCreated  = "ASSET_CREATED"
	EventAssetUpdated  = "ASSET_UPDATED"
	EventAssetDeleted  = "ASSET_DELETED"
	EventAssetShared   = "ASSET_SHARED"
)

// Event represents a generic domain event
type Event struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

// EventPublisher abstracts the message broker for publishing events
type EventPublisher interface {
	PublishTeamEvent(ctx context.Context, eventType string, payload interface{}) error
	PublishAssetEvent(ctx context.Context, eventType string, payload interface{}) error
}
