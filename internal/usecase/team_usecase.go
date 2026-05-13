package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/seta-training/internal/domain"
)

type teamUseCase struct {
	teamRepo  domain.TeamRepository
	userRepo  domain.UserRepository
	cache     domain.Cache
	publisher domain.EventPublisher
}

func NewTeamUseCase(teamRepo domain.TeamRepository, userRepo domain.UserRepository, cache domain.Cache, publisher domain.EventPublisher) domain.TeamUseCase {
	return &teamUseCase{
		teamRepo:  teamRepo,
		userRepo:  userRepo,
		cache:     cache,
		publisher: publisher,
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

	// Publish Event
	_ = u.publisher.PublishTeamEvent(ctx, domain.EventTeamCreated, map[string]interface{}{
		"teamId":   team.ID,
		"teamName": team.Name,
		"ownerId":  requesterID,
	})

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

	err = u.teamRepo.AddMember(ctx, teamID, userID)
	if err == nil {
		_ = u.publisher.PublishTeamEvent(ctx, domain.EventMemberAdded, map[string]interface{}{
			"teamId": teamID,
			"userId": userID,
		})
		_ = u.cache.Delete(ctx, u.memberTeamsCacheKey(userID))
	}
	return err
}

func (u *teamUseCase) memberTeamsCacheKey(userID uint) string {
	return fmt.Sprintf("user_teams:%d", userID)
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

	err = u.teamRepo.RemoveMember(ctx, teamID, userID)
	if err == nil {
		_ = u.publisher.PublishTeamEvent(ctx, domain.EventMemberRemoved, map[string]interface{}{
			"teamId": teamID,
			"userId": userID,
		})
		// invalidation
		_ = u.cache.Delete(ctx, u.memberTeamsCacheKey(userID))
	}
	return err
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

	err = u.teamRepo.AddManager(ctx, teamID, userID)
	if err == nil {
		_ = u.publisher.PublishTeamEvent(ctx, domain.EventMemberAdded, map[string]interface{}{
			"teamId": teamID,
			"userId": userID,
			"role":   "manager",
		})
		_ = u.cache.Delete(ctx, u.memberTeamsCacheKey(userID))
	}
	return err
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

	err = u.teamRepo.RemoveManager(ctx, teamID, userID)
	if err == nil {
		_ = u.publisher.PublishTeamEvent(ctx, domain.EventMemberRemoved, map[string]interface{}{
			"teamId": teamID,
			"userId": userID,
			"role":   "manager",
		})
		_ = u.cache.Delete(ctx, u.memberTeamsCacheKey(userID))
	}
	return err
}
