package repository

import (
	"context"

	"github.com/seta-training/core/internal/domain"
	"gorm.io/gorm"
)

type assetRepository struct {
	db *gorm.DB
}

func NewAssetRepository(db *gorm.DB) domain.AssetRepository {
	return &assetRepository{db: db}
}

func (r *assetRepository) CreateFolder(ctx context.Context, folder *domain.Folder) error {
	return r.db.WithContext(ctx).Create(folder).Error
}

func (r *assetRepository) GetFolderByID(ctx context.Context, id uint) (*domain.Folder, error) {
	var folder domain.Folder
	err := r.db.WithContext(ctx).Preload("Notes").First(&folder, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &folder, nil
}

func (r *assetRepository) CreateNote(ctx context.Context, note *domain.Note) error {
	return r.db.WithContext(ctx).Create(note).Error
}

func (r *assetRepository) GetNoteByID(ctx context.Context, id uint) (*domain.Note, error) {
	var note domain.Note
	err := r.db.WithContext(ctx).First(&note, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &note, nil
}

func (r *assetRepository) UpdateNote(ctx context.Context, note *domain.Note) error {
	return r.db.WithContext(ctx).Save(note).Error
}

func (r *assetRepository) DeleteNote(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&domain.Note{}, id).Error
}

func (r *assetRepository) SetPermission(ctx context.Context, perm *domain.AssetPermission) error {
	// Upsert permission
	var existing domain.AssetPermission
	err := r.db.WithContext(ctx).Where("asset_type = ? AND asset_id = ? AND user_id = ?", perm.AssetType, perm.AssetID, perm.UserID).First(&existing).Error
	
	if err == nil {
		existing.AccessLevel = perm.AccessLevel
		return r.db.WithContext(ctx).Save(&existing).Error
	}
	
	if err == gorm.ErrRecordNotFound {
		return r.db.WithContext(ctx).Create(perm).Error
	}
	return err
}

func (r *assetRepository) RemovePermission(ctx context.Context, assetType domain.AssetType, assetID, userID uint) error {
	return r.db.WithContext(ctx).Where("asset_type = ? AND asset_id = ? AND user_id = ?", assetType, assetID, userID).Delete(&domain.AssetPermission{}).Error
}

func (r *assetRepository) GetPermission(ctx context.Context, assetType domain.AssetType, assetID, userID uint) (*domain.AssetPermission, error) {
	var perm domain.AssetPermission
	err := r.db.WithContext(ctx).Where("asset_type = ? AND asset_id = ? AND user_id = ?", assetType, assetID, userID).First(&perm).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &perm, nil
}
