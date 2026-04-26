package usecase

import (
	"context"
	"errors"

	"github.com/seta-training/core/internal/domain"
)

type teamUseCase struct {
	teamRepo domain.TeamRepository
	userRepo domain.UserRepository
}

func NewTeamUseCase(teamRepo domain.TeamRepository, userRepo domain.UserRepository) domain.TeamUseCase {
	return &teamUseCase{
		teamRepo: teamRepo,
		userRepo: userRepo,
	}
}

func (u *teamUseCase) isManager(team *domain.Team, userID uint) bool {
	for _, mgr := range team.Managers {
		if mgr.ID == userID {
			return true
		}
	}
	return false
}

func (u *teamUseCase) CreateTeam(ctx context.Context, req *domain.CreateTeamRequest, requesterID uint) (*domain.Team, error) {
	// Only managers can create teams
	requester, err := u.userRepo.GetByID(ctx, requesterID)
	if err != nil {
		return nil, err
	}
	if requester == nil || requester.Role != domain.RoleManager {
		return nil, errors.New("only managers can create teams")
	}

	team := &domain.Team{
		Name:     req.Name,
		Managers: []*domain.User{requester}, // Creator is the first manager
	}

	if err := u.teamRepo.Create(ctx, team); err != nil {
		return nil, err
	}

	return team, nil
}

func (u *teamUseCase) AddMember(ctx context.Context, teamID, userID, requesterID uint) error {
	team, err := u.teamRepo.GetByID(ctx, teamID)
	if err != nil {
		return err
	}
	if team == nil {
		return errors.New("team not found")
	}

	if !u.isManager(team, requesterID) {
		return errors.New("only team managers can add members")
	}

	// Verify user exists
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	return u.teamRepo.AddMember(ctx, teamID, userID)
}

func (u *teamUseCase) RemoveMember(ctx context.Context, teamID, userID, requesterID uint) error {
	team, err := u.teamRepo.GetByID(ctx, teamID)
	if err != nil {
		return err
	}
	if team == nil {
		return errors.New("team not found")
	}

	if !u.isManager(team, requesterID) {
		return errors.New("only team managers can remove members")
	}

	return u.teamRepo.RemoveMember(ctx, teamID, userID)
}

func (u *teamUseCase) AddManager(ctx context.Context, teamID, userID, requesterID uint) error {
	team, err := u.teamRepo.GetByID(ctx, teamID)
	if err != nil {
		return err
	}
	if team == nil {
		return errors.New("team not found")
	}

	if len(team.Managers) == 0 || team.Managers[0].ID != requesterID {
		return errors.New("only the main manager can add other managers")
	}

	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}
	if user.Role != domain.RoleManager {
		return errors.New("only users with 'manager' role can be added as team managers")
	}

	return u.teamRepo.AddManager(ctx, teamID, userID)
}

func (u *teamUseCase) RemoveManager(ctx context.Context, teamID, userID, requesterID uint) error {
	team, err := u.teamRepo.GetByID(ctx, teamID)
	if err != nil {
		return err
	}
	if team == nil {
		return errors.New("team not found")
	}

	if len(team.Managers) == 0 || team.Managers[0].ID != requesterID {
		return errors.New("only the main manager can remove other managers")
	}

	if userID == requesterID {
		return errors.New("main manager cannot remove themselves")
	}

	return u.teamRepo.RemoveManager(ctx, teamID, userID)
}
