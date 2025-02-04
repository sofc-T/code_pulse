package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/sofc-t/code_pulse/dto"
	"github.com/sofc-t/code_pulse/infrastructure/user"
	"github.com/sofc-t/code_pulse/delivery/core"
)

type LoginController interface {
	Login(ctx *gin.Context) string
}

type loginController struct {
	loginService service.LoginService
	jWtService   core.JWTService
}

func NewLoginController(loginService service.LoginService,
	jWtService core.JWTService) LoginController {
	return &loginController{
		loginService: loginService,
		jWtService:   jWtService,
	}
}

func (controller *loginController) Login(ctx *gin.Context) string {
	var credentials dto.Login
	err := ctx.ShouldBind(&credentials)
	if err != nil {
		return ""
	}
	isAuthenticated := controller.loginService.Login(credentials.Email, credentials.Password)
	if isAuthenticated {
		return controller.jWtService.GenerateToken(credentials.Email, true)
	}
	return ""
}
