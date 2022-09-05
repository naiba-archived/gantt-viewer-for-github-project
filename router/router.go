package router

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/naiba/gantt-viewer-for-github-project/model"
	"github.com/naiba/gantt-viewer-for-github-project/singleton"
)

func UserAuthorize(c *fiber.Ctx) error {
	sid := strings.TrimSpace(c.Cookies(model.KeyAuthCookie))
	if sid != "" {
		var user model.User
		if err := singleton.GetDB().Where("sid = ?", sid).First(&user).Error; err != nil {
			return err
		}
		c.Locals(model.KeyAuthorizedUser, &user)
	}
	return c.Next()
}

func LoginRequired(c *fiber.Ctx) error {
	user := c.Locals(model.KeyAuthorizedUser)
	if user == nil {
		return errors.New("login required")
	}
	return c.Next()
}

func AnonymousRequired(c *fiber.Ctx) error {
	user := c.Locals(model.KeyAuthorizedUser)
	if user != nil {
		return errors.New("already login")
	}
	return c.Next()
}
