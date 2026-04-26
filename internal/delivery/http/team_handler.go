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

// CreateTeam godoc
// @Summary Create a new team
// @Description Allows a manager to create a new team
// @Tags teams
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body domain.CreateTeamRequest true "Team details"
// @Success 201 {object} domain.Team
// @Failure 400 {object} map[string]string
// @Router /teams [post]
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

// AddMember godoc
// @Summary Add a member to a team
// @Description Allows a team manager to add a member to their team
// @Tags teams
// @Security Bearer
// @Accept json
// @Produce json
// @Param id path int true "Team ID"
// @Param request body map[string]int true "User ID"
// @Success 200
// @Failure 400 {object} map[string]string
// @Router /teams/{id}/members [post]
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

// RemoveMember godoc
// @Summary Remove a member from a team
// @Description Allows a team manager to remove a member from their team
// @Tags teams
// @Security Bearer
// @Param id path int true "Team ID"
// @Param userId path int true "User ID to remove"
// @Success 200
// @Failure 400 {object} map[string]string
// @Router /teams/{id}/members/{userId} [delete]
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

// AddManager godoc
// @Summary Add a manager to a team
// @Description Allows the main manager to add another manager to the team
// @Tags teams
// @Security Bearer
// @Accept json
// @Produce json
// @Param id path int true "Team ID"
// @Param request body map[string]int true "User ID"
// @Success 200
// @Failure 400 {object} map[string]string
// @Router /teams/{id}/managers [post]
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

// RemoveManager godoc
// @Summary Remove a manager from a team
// @Description Allows the main manager to remove another manager from the team
// @Tags teams
// @Security Bearer
// @Param id path int true "Team ID"
// @Param userId path int true "User ID to remove"
// @Success 200
// @Failure 400 {object} map[string]string
// @Router /teams/{id}/managers/{userId} [delete]
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
