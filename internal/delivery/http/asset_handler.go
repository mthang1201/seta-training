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

// CreateFolder godoc
// @Summary Create a new folder
// @Description Create a new digital folder for assets
// @Tags assets
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body map[string]string true "Folder details"
// @Success 201 {object} domain.Folder
// @Failure 400 {object} map[string]string
// @Router /assets/folders [post]
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

// GetFolder godoc
// @Summary Get folder details
// @Description Retrieve a folder and its notes, checking access permissions
// @Tags assets
// @Security Bearer
// @Produce json
// @Param id path int true "Folder ID"
// @Success 200 {object} domain.Folder
// @Failure 400,403 {object} map[string]string
// @Router /assets/folders/{id} [get]
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

// CreateNote godoc
// @Summary Create a note in a folder
// @Description Create a new note within a specified folder
// @Tags assets
// @Security Bearer
// @Accept json
// @Produce json
// @Param id path int true "Folder ID"
// @Param request body map[string]string true "Note details"
// @Success 201 {object} domain.Note
// @Failure 400,403 {object} map[string]string
// @Router /assets/folders/{id}/notes [post]
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

// GetNote godoc
// @Summary Get note details
// @Description Retrieve a specific note, checking access permissions
// @Tags assets
// @Security Bearer
// @Produce json
// @Param id path int true "Note ID"
// @Success 200 {object} domain.Note
// @Failure 400,403 {object} map[string]string
// @Router /assets/notes/{id} [get]
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

// UpdateNote godoc
// @Summary Update a note
// @Description Update the title or content of an existing note
// @Tags assets
// @Security Bearer
// @Accept json
// @Produce json
// @Param id path int true "Note ID"
// @Param request body map[string]string true "Note updates"
// @Success 200 {object} domain.Note
// @Failure 400,403 {object} map[string]string
// @Router /assets/notes/{id} [put]
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

// ShareAsset godoc
// @Summary Share an asset
// @Description Share a folder or note with another user
// @Tags assets
// @Security Bearer
// @Accept json
// @Produce json
// @Param type path string true "Asset Type (folder/note)"
// @Param id path int true "Asset ID"
// @Param request body map[string]interface{} true "Sharing details"
// @Success 200
// @Failure 400,403 {object} map[string]string
// @Router /assets/{type}/{id}/share [post]
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

// RevokeAccess godoc
// @Summary Revoke access to an asset
// @Description Revoke a user's access to a folder or note
// @Tags assets
// @Security Bearer
// @Param type path string true "Asset Type (folder/note)"
// @Param id path int true "Asset ID"
// @Param targetUserId path int true "User ID to revoke"
// @Success 200
// @Failure 400,403 {object} map[string]string
// @Router /assets/{type}/{id}/share/{targetUserId} [delete]
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
