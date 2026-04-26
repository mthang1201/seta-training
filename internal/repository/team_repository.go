package repository

import (
	"context"

	"github.com/seta-training/core/internal/domain"
	"gorm.io/gorm"
)

type teamRepository struct {
	db *gorm.DB
}

func NewTeamRepository(db *gorm.DB) domain.TeamRepository {
	return &teamRepository{db: db}
}

func (r *teamRepository) Create(ctx context.Context, team *domain.Team) error {
	return r.db.WithContext(ctx).Create(team).Error
}

func (r *teamRepository) GetByID(ctx context.Context, id uint) (*domain.Team, error) {
	var team domain.Team
	err := r.db.WithContext(ctx).Preload("Managers").Preload("Members").First(&team, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &team, nil
}

func (r *teamRepository) AddManager(ctx context.Context, teamID, userID uint) error {
	team := &domain.Team{ID: teamID}
	user := &domain.User{ID: userID}
	return r.db.WithContext(ctx).Model(team).Association("Managers").Append(user)
}

func (r *teamRepository) RemoveManager(ctx context.Context, teamID, userID uint) error {
	team := &domain.Team{ID: teamID}
	user := &domain.User{ID: userID}
	return r.db.WithContext(ctx).Model(team).Association("Managers").Delete(user)
}

func (r *teamRepository) AddMember(ctx context.Context, teamID, userID uint) error {
	team := &domain.Team{ID: teamID}
	user := &domain.User{ID: userID}
	return r.db.WithContext(ctx).Model(team).Association("Members").Append(user)
}

func (r *teamRepository) RemoveMember(ctx context.Context, teamID, userID uint) error {
	team := &domain.Team{ID: teamID}
	user := &domain.User{ID: userID}
	return r.db.WithContext(ctx).Model(team).Association("Members").Delete(user)
}

func (r *teamRepository) GetTeamsByMemberID(ctx context.Context, userID uint) ([]*domain.Team, error) {
	var teams []*domain.Team
	err := r.db.WithContext(ctx).
		Joins("JOIN team_members on team_members.team_id = teams.id").
		Where("team_members.user_id = ?", userID).
		Preload("Managers").
		Find(&teams).Error
	return teams, err
}
