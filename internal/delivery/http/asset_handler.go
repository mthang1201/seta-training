package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/seta-training/core/internal/config"
	"github.com/seta-training/core/internal/domain"
)

type AssetHandler struct {
	assetUseCase domain.AssetUseCase
}

func NewAssetHandler(r *gin.RouterGroup, useCase domain.AssetUseCase, cfg *config.Config) {
	handler := &AssetHandler{
		assetUseCase: useCase,
	}

	assets := r.Group("/assets")
	assets.Use(AuthMiddleware(cfg))

	assets.POST("/folders", handler.CreateFolder)
	assets.GET("/folders/:id", handler.GetFolder)
	
	assets.POST("/folders/:id/notes", handler.CreateNote)
	assets.GET("/notes/:id", handler.GetNote)
	assets.PUT("/notes/:id", handler.UpdateNote)
	
	assets.POST("/:type/:id/share", handler.ShareAsset)
	assets.DELETE("/:type/:id/share/:targetUserId", handler.RevokeAccess)
}

func (h *AssetHandler) getRequesterID(c *gin.Context) uint {
	idFloat, exists := c.Get("userId")
	if !exists {
		return 0
	}
	return uint(idFloat.(float64))
}

func (h *AssetHandler) CreateFolder(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	requesterID := h.getRequesterID(c)

	folder, err := h.assetUseCase.CreateFolder(c.Request.Context(), req.Name, requesterID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, folder)
}

func (h *AssetHandler) GetFolder(c *gin.Context) {
	folderID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder ID"})
		return
	}

	requesterID := h.getRequesterID(c)

	folder, err := h.assetUseCase.GetFolder(c.Request.Context(), uint(folderID), requesterID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, folder)
}

func (h *AssetHandler) CreateNote(c *gin.Context) {
	folderID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder ID"})
		return
	}

	var req struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	requesterID := h.getRequesterID(c)

	note, err := h.assetUseCase.CreateNote(c.Request.Context(), uint(folderID), req.Title, req.Content, requesterID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, note)
}

func (h *AssetHandler) GetNote(c *gin.Context) {
	noteID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid note ID"})
		return
	}

	requesterID := h.getRequesterID(c)

	note, err := h.assetUseCase.GetNote(c.Request.Context(), uint(noteID), requesterID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, note)
}

func (h *AssetHandler) UpdateNote(c *gin.Context) {
	noteID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid note ID"})
		return
	}

	var req struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	requesterID := h.getRequesterID(c)

	note, err := h.assetUseCase.UpdateNote(c.Request.Context(), uint(noteID), req.Title, req.Content, requesterID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, note)
}

func (h *AssetHandler) ShareAsset(c *gin.Context) {
	assetType := domain.AssetType(c.Param("type"))
	if assetType != domain.AssetFolder && assetType != domain.AssetNote {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid asset type"})
		return
	}

	assetID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid asset ID"})
		return
	}

	var req struct {
		TargetUserID uint               `json:"targetUserId" binding:"required"`
		AccessLevel  domain.AccessLevel `json:"accessLevel" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	requesterID := h.getRequesterID(c)

	shareReq := &domain.ShareAssetRequest{
		AssetType:    assetType,
		AssetID:      uint(assetID),
		TargetUserID: req.TargetUserID,
		AccessLevel:  req.AccessLevel,
	}

	if err := h.assetUseCase.ShareAsset(c.Request.Context(), shareReq, requesterID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func (h *AssetHandler) RevokeAccess(c *gin.Context) {
	assetType := domain.AssetType(c.Param("type"))
	if assetType != domain.AssetFolder && assetType != domain.AssetNote {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid asset type"})
		return
	}

	assetID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid asset ID"})
		return
	}

	targetUserID, err := strconv.ParseUint(c.Param("targetUserId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target user ID"})
		return
	}

	requesterID := h.getRequesterID(c)

	if err := h.assetUseCase.RevokeAccess(c.Request.Context(), assetType, uint(assetID), uint(targetUserID), requesterID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}
