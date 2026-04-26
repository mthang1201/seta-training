package domain

import (
	"context"
	"time"
)

type Team struct {
	ID        uint      `json:"teamId" gorm:"primaryKey"`
	Name      string    `json:"teamName" gorm:"not null"`
	Managers  []*User   `json:"managers" gorm:"many2many:team_managers;"`
	Members   []*User   `json:"members" gorm:"many2many:team_members;"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type TeamRepository interface {
	Create(ctx context.Context, team *Team) error
	GetByID(ctx context.Context, id uint) (*Team, error)
	AddManager(ctx context.Context, teamID, userID uint) error
	RemoveManager(ctx context.Context, teamID, userID uint) error
	AddMember(ctx context.Context, teamID, userID uint) error
	RemoveMember(ctx context.Context, teamID, userID uint) error
	GetTeamsByMemberID(ctx context.Context, userID uint) ([]*Team, error)
}

type TeamUseCase interface {
	CreateTeam(ctx context.Context, req *CreateTeamRequest, requesterID uint) (*Team, error)
	AddMember(ctx context.Context, teamID, userID, requesterID uint) error
	RemoveMember(ctx context.Context, teamID, userID, requesterID uint) error
	AddManager(ctx context.Context, teamID, userID, requesterID uint) error
	RemoveManager(ctx context.Context, teamID, userID, requesterID uint) error
}

type CreateTeamRequest struct {
	Name string `json:"teamName"`
}
