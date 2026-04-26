package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/seta-training/core/internal/config"
	"github.com/seta-training/core/internal/domain"
)

type UserHandler struct {
	userUseCase domain.UserUseCase
}

func NewUserHandler(r *gin.RouterGroup, useCase domain.UserUseCase, cfg *config.Config) {
	handler := &UserHandler{
		userUseCase: useCase,
	}

	auth := r.Group("/auth")
	auth.POST("/register", handler.Register)
	auth.POST("/login", handler.Login)

	users := r.Group("/users")
	users.Use(AuthMiddleware(cfg))
	users.GET("/", handler.GetUsers) // Requires Auth
	
	// Assuming bulk import requires manager role
	users.POST("/import", RoleMiddleware(string(domain.RoleManager)), handler.ImportUsers)
}

// Register godoc
// @Summary Register a new user
// @Description Create a new user with a specific role
// @Tags auth
// @Accept json
// @Produce json
// @Param request body domain.RegisterRequest true "Registration Info"
// @Success 201 {object} domain.User
// @Failure 400 {object} map[string]string
// @Router /auth/register [post]
func (h *UserHandler) Register(c *gin.Context) {
	var req domain.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	user, err := h.userUseCase.Register(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// Login godoc
// @Summary Login a user
// @Description Authenticate a user and return a JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body domain.LoginRequest true "Login Credentials"
// @Success 200 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /auth/login [post]
func (h *UserHandler) Login(c *gin.Context) {
	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	token, err := h.userUseCase.Login(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

// GetUsers godoc
// @Summary Get all users
// @Description Retrieve a list of all registered users
// @Tags users
// @Security Bearer
// @Produce json
// @Success 200 {array} domain.User
// @Failure 500 {object} map[string]string
// @Router /users [get]
func (h *UserHandler) GetUsers(c *gin.Context) {
	users, err := h.userUseCase.GetUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

// ImportUsers godoc
// @Summary Bulk import users from CSV
// @Description Concurrently create users from an uploaded CSV file
// @Tags users
// @Security Bearer
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "CSV file with columns: username, email, password, role"
// @Success 200 {object} domain.ImportResult
// @Failure 400 {object} map[string]string
// @Router /users/import [post]
func (h *UserHandler) ImportUsers(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get file from form: 'file' is required"})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
		return
	}
	defer f.Close()

	result, err := h.userUseCase.ImportUsers(c.Request.Context(), f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
