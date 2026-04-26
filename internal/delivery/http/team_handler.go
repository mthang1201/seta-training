package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/seta-training/core/internal/config"
	"github.com/seta-training/core/internal/domain"
)

type TeamHandler struct {
	teamUseCase domain.TeamUseCase
}

func NewTeamHandler(r *gin.RouterGroup, useCase domain.TeamUseCase, cfg *config.Config) {
	handler := &TeamHandler{
		teamUseCase: useCase,
	}

	teams := r.Group("/teams")
	teams.Use(AuthMiddleware(cfg))
	teams.Use(RoleMiddleware(string(domain.RoleManager))) // Only managers can interact with team endpoints

	teams.POST("/", handler.CreateTeam)
	teams.POST("/:id/members", handler.AddMember)
	teams.DELETE("/:id/members/:userId", handler.RemoveMember)
	teams.POST("/:id/managers", handler.AddManager)
	teams.DELETE("/:id/managers/:userId", handler.RemoveManager)
}

func (h *TeamHandler) getRequesterID(c *gin.Context) uint {
	idFloat, exists := c.Get("userId")
	if !exists {
		return 0
	}
	return uint(idFloat.(float64))
}

func (h *TeamHandler) CreateTeam(c *gin.Context) {
	var req domain.CreateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	requesterID := h.getRequesterID(c)

	team, err := h.teamUseCase.CreateTeam(c.Request.Context(), &req, requesterID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, team)
}

func (h *TeamHandler) AddMember(c *gin.Context) {
	teamID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid team ID"})
		return
	}

	var req struct {
		UserID uint `json:"userId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	requesterID := h.getRequesterID(c)

	if err := h.teamUseCase.AddMember(c.Request.Context(), uint(teamID), req.UserID, requesterID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func (h *TeamHandler) RemoveMember(c *gin.Context) {
	teamID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid team ID"})
		return
	}
	userID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	requesterID := h.getRequesterID(c)

	if err := h.teamUseCase.RemoveMember(c.Request.Context(), uint(teamID), uint(userID), requesterID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func (h *TeamHandler) AddManager(c *gin.Context) {
	teamID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid team ID"})
		return
	}

	var req struct {
		UserID uint `json:"userId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	requesterID := h.getRequesterID(c)

	if err := h.teamUseCase.AddManager(c.Request.Context(), uint(teamID), req.UserID, requesterID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func (h *TeamHandler) RemoveManager(c *gin.Context) {
	teamID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid team ID"})
		return
	}
	userID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	requesterID := h.getRequesterID(c)

	if err := h.teamUseCase.RemoveManager(c.Request.Context(), uint(teamID), uint(userID), requesterID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}
