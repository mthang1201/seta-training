package usecase

import (
	"context"
	"errors"

	"github.com/seta-training/core/internal/domain"
)

type assetUseCase struct {
	assetRepo domain.AssetRepository
	teamRepo  domain.TeamRepository
}

func NewAssetUseCase(assetRepo domain.AssetRepository, teamRepo domain.TeamRepository) domain.AssetUseCase {
	return &assetUseCase{assetRepo: assetRepo, teamRepo: teamRepo}
}

// Check if requester has read access
func (u *assetUseCase) canReadAsset(ctx context.Context, assetType domain.AssetType, assetID uint, ownerID uint, requesterID uint) (bool, error) {
	if requesterID == ownerID {
		return true, nil
	}

	// Direct permission
	perm, err := u.assetRepo.GetPermission(ctx, assetType, assetID, requesterID)
	if err != nil {
		return false, err
	}
	if perm != nil {
		return true, nil
	}

	// Inheritance for notes
	if assetType == domain.AssetNote {
		note, err := u.assetRepo.GetNoteByID(ctx, assetID)
		if err != nil {
			return false, err
		}
		if note != nil {
			folderPerm, err := u.assetRepo.GetPermission(ctx, domain.AssetFolder, note.FolderID, requesterID)
			if err != nil {
				return false, err
			}
			if folderPerm != nil {
				return true, nil
			}
		}
	}

	// Manager Oversight
	// Managers have read-only access to assets owned by their team members.
	teams, err := u.teamRepo.GetTeamsByMemberID(ctx, ownerID)
	if err != nil {
		return false, err
	}
	for _, team := range teams {
		for _, mgr := range team.Managers {
			if mgr.ID == requesterID {
				return true, nil // Manager found
			}
		}
	}

	return false, nil
}

// Check if requester has write access
func (u *assetUseCase) canWriteAsset(ctx context.Context, assetType domain.AssetType, assetID uint, ownerID uint, requesterID uint) (bool, error) {
	if requesterID == ownerID {
		return true, nil
	}

	// Direct permission
	perm, err := u.assetRepo.GetPermission(ctx, assetType, assetID, requesterID)
	if err != nil {
		return false, err
	}
	if perm != nil && perm.AccessLevel == domain.AccessWrite {
		return true, nil
	}

	// Inheritance for notes
	if assetType == domain.AssetNote {
		note, err := u.assetRepo.GetNoteByID(ctx, assetID)
		if err != nil {
			return false, err
		}
		if note != nil {
			folderPerm, err := u.assetRepo.GetPermission(ctx, domain.AssetFolder, note.FolderID, requesterID)
			if err != nil {
				return false, err
			}
			if folderPerm != nil && folderPerm.AccessLevel == domain.AccessWrite {
				return true, nil
			}
		}
	}

	return false, nil
}

func (u *assetUseCase) CreateFolder(ctx context.Context, name string, ownerID uint) (*domain.Folder, error) {
	folder := &domain.Folder{
		Name:    name,
		OwnerID: ownerID,
	}
	if err := u.assetRepo.CreateFolder(ctx, folder); err != nil {
		return nil, err
	}
	return folder, nil
}

func (u *assetUseCase) CreateNote(ctx context.Context, folderID uint, title, content string, requesterID uint) (*domain.Note, error) {
	folder, err := u.assetRepo.GetFolderByID(ctx, folderID)
	if err != nil {
		return nil, err
	}
	if folder == nil {
		return nil, errors.New("folder not found")
	}

	canWrite, err := u.canWriteAsset(ctx, domain.AssetFolder, folderID, folder.OwnerID, requesterID)
	if err != nil {
		return nil, err
	}
	if !canWrite {
		return nil, errors.New("forbidden: write access required")
	}

	note := &domain.Note{
		FolderID: folderID,
		Title:    title,
		Content:  content,
	}
	if err := u.assetRepo.CreateNote(ctx, note); err != nil {
		return nil, err
	}
	return note, nil
}

func (u *assetUseCase) GetFolder(ctx context.Context, folderID uint, requesterID uint) (*domain.Folder, error) {
	folder, err := u.assetRepo.GetFolderByID(ctx, folderID)
	if err != nil {
		return nil, err
	}
	if folder == nil {
		return nil, errors.New("folder not found")
	}

	canRead, err := u.canReadAsset(ctx, domain.AssetFolder, folderID, folder.OwnerID, requesterID)
	if err != nil {
		return nil, err
	}
	if !canRead {
		return nil, errors.New("forbidden: read access required")
	}

	return folder, nil
}

func (u *assetUseCase) GetNote(ctx context.Context, noteID uint, requesterID uint) (*domain.Note, error) {
	note, err := u.assetRepo.GetNoteByID(ctx, noteID)
	if err != nil {
		return nil, err
	}
	if note == nil {
		return nil, errors.New("note not found")
	}

	folder, err := u.assetRepo.GetFolderByID(ctx, note.FolderID)
	if err != nil {
		return nil, err
	}

	canRead, err := u.canReadAsset(ctx, domain.AssetNote, noteID, folder.OwnerID, requesterID)
	if err != nil {
		return nil, err
	}
	if !canRead {
		return nil, errors.New("forbidden: read access required")
	}

	return note, nil
}

func (u *assetUseCase) UpdateNote(ctx context.Context, noteID uint, title, content string, requesterID uint) (*domain.Note, error) {
	note, err := u.assetRepo.GetNoteByID(ctx, noteID)
	if err != nil {
		return nil, err
	}
	if note == nil {
		return nil, errors.New("note not found")
	}

	folder, err := u.assetRepo.GetFolderByID(ctx, note.FolderID)
	if err != nil {
		return nil, err
	}

	canWrite, err := u.canWriteAsset(ctx, domain.AssetNote, noteID, folder.OwnerID, requesterID)
	if err != nil {
		return nil, err
	}
	if !canWrite {
		return nil, errors.New("forbidden: write access required")
	}

	note.Title = title
	note.Content = content
	if err := u.assetRepo.UpdateNote(ctx, note); err != nil {
		return nil, err
	}

	return note, nil
}

func (u *assetUseCase) ShareAsset(ctx context.Context, req *domain.ShareAssetRequest, requesterID uint) error {
	var ownerID uint
	if req.AssetType == domain.AssetFolder {
		folder, err := u.assetRepo.GetFolderByID(ctx, req.AssetID)
		if err != nil { return err }
		if folder == nil { return errors.New("folder not found") }
		ownerID = folder.OwnerID
	} else if req.AssetType == domain.AssetNote {
		note, err := u.assetRepo.GetNoteByID(ctx, req.AssetID)
		if err != nil { return err }
		if note == nil { return errors.New("note not found") }
		folder, err := u.assetRepo.GetFolderByID(ctx, note.FolderID)
		if err != nil { return err }
		ownerID = folder.OwnerID
	} else {
		return errors.New("invalid asset type")
	}

	if requesterID != ownerID {
		return errors.New("only owner can share asset")
	}

	perm := &domain.AssetPermission{
		AssetType:   req.AssetType,
		AssetID:     req.AssetID,
		UserID:      req.TargetUserID,
		AccessLevel: req.AccessLevel,
	}

	return u.assetRepo.SetPermission(ctx, perm)
}

func (u *assetUseCase) RevokeAccess(ctx context.Context, assetType domain.AssetType, assetID, targetUserID, requesterID uint) error {
	var ownerID uint
	if assetType == domain.AssetFolder {
		folder, err := u.assetRepo.GetFolderByID(ctx, assetID)
		if err != nil { return err }
		if folder == nil { return errors.New("folder not found") }
		ownerID = folder.OwnerID
	} else if assetType == domain.AssetNote {
		note, err := u.assetRepo.GetNoteByID(ctx, assetID)
		if err != nil { return err }
		if note == nil { return errors.New("note not found") }
		folder, err := u.assetRepo.GetFolderByID(ctx, note.FolderID)
		if err != nil { return err }
		ownerID = folder.OwnerID
	} else {
		return errors.New("invalid asset type")
	}

	if requesterID != ownerID {
		return errors.New("only owner can revoke access")
	}

	return u.assetRepo.RemovePermission(ctx, assetType, assetID, targetUserID)
}
