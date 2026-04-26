package domain

import (
	"context"
	"time"
)

type AccessLevel string

const (
	AccessRead  AccessLevel = "read"
	AccessWrite AccessLevel = "write"
)

type AssetType string

const (
	AssetFolder AssetType = "folder"
	AssetNote   AssetType = "note"
)

type Folder struct {
	ID        uint      `json:"folderId" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"not null"`
	OwnerID   uint      `json:"ownerId" gorm:"not null"`
	Notes     []Note    `json:"notes,omitempty" gorm:"foreignKey:FolderID;constraint:OnDelete:CASCADE"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Note struct {
	ID        uint      `json:"noteId" gorm:"primaryKey"`
	FolderID  uint      `json:"folderId" gorm:"not null"`
	Title     string    `json:"title" gorm:"not null"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type AssetPermission struct {
	ID          uint        `json:"id" gorm:"primaryKey"`
	AssetType   AssetType   `json:"assetType" gorm:"not null;index:idx_asset_user"`
	AssetID     uint        `json:"assetId" gorm:"not null;index:idx_asset_user"`
	UserID      uint        `json:"userId" gorm:"not null;index:idx_asset_user"`
	AccessLevel AccessLevel `json:"accessLevel" gorm:"not null"`
	CreatedAt   time.Time   `json:"createdAt"`
}

type AssetRepository interface {
	CreateFolder(ctx context.Context, folder *Folder) error
	GetFolderByID(ctx context.Context, id uint) (*Folder, error)
	CreateNote(ctx context.Context, note *Note) error
	GetNoteByID(ctx context.Context, id uint) (*Note, error)
	UpdateNote(ctx context.Context, note *Note) error
	DeleteNote(ctx context.Context, id uint) error

	// Permissions
	SetPermission(ctx context.Context, perm *AssetPermission) error
	RemovePermission(ctx context.Context, assetType AssetType, assetID, userID uint) error
	GetPermission(ctx context.Context, assetType AssetType, assetID, userID uint) (*AssetPermission, error)
}

type AssetUseCase interface {
	CreateFolder(ctx context.Context, name string, ownerID uint) (*Folder, error)
	CreateNote(ctx context.Context, folderID uint, title, content string, requesterID uint) (*Note, error)
	GetFolder(ctx context.Context, folderID uint, requesterID uint) (*Folder, error)
	GetNote(ctx context.Context, noteID uint, requesterID uint) (*Note, error)
	UpdateNote(ctx context.Context, noteID uint, title, content string, requesterID uint) (*Note, error)
	
	ShareAsset(ctx context.Context, req *ShareAssetRequest, requesterID uint) error
	RevokeAccess(ctx context.Context, assetType AssetType, assetID, targetUserID, requesterID uint) error
}

type ShareAssetRequest struct {
	AssetType   AssetType   `json:"assetType" binding:"required"`
	AssetID     uint        `json:"assetId" binding:"required"`
	TargetUserID uint       `json:"targetUserId" binding:"required"`
	AccessLevel AccessLevel `json:"accessLevel" binding:"required"`
}
