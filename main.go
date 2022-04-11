package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/html"
	"github.com/patrickmn/go-cache"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"

	"github.com/naiba/gantt-viewer-for-github-project/model"
	"github.com/naiba/gantt-viewer-for-github-project/singleton"
	"github.com/naiba/gantt-viewer-for-github-project/util"
)

func main() {
	engine := html.New("./views", ".html")
	engine.Reload(true)
	engine.Debug(true)

	app := fiber.New(
		fiber.Config{
			Views: engine,
		},
	)
	app.Use(recover.New())
	app.Use(userAuthorize)

	app.Get("/", func(c *fiber.Ctx) error {
		user := c.Locals(model.KeyAuthorizedUser)
		name := "World"
		if u, _ := user.(*model.User); u != nil {
			name = u.GitHubLogin
		}
		return c.SendString(fmt.Sprintf("Hello, %s 👋!", name))
	})

	oauth2group := app.Group("/oauth2")
	{
		oauth2group.Get("/login", func(c *fiber.Ctx) error {
			state, err := util.GenerateRandomString(16)
			if err != nil {
				return err
			}
			c.Cookie(&fiber.Cookie{
				Name:     "oa_state",
				Value:    state,
				Expires:  time.Now().Add(time.Second * 60 * 10),
				Secure:   false,
				HTTPOnly: false,
			})

			singleton.Cache.Set(fmt.Sprintf("os::%s", c.IP()), state, cache.DefaultExpiration)
			return c.Render("redirect", map[string]interface{}{
				"URL": singleton.GetOauth2Config().AuthCodeURL(state, oauth2.AccessTypeOnline),
			})
		})
		oauth2group.Get("/callback", func(c *fiber.Ctx) error {
			state, ok := singleton.Cache.Get(fmt.Sprintf("os::%s", c.IP()))
			stateFromCookie := c.Cookies("oa_state")
			if !ok || state.(string) != c.Query("state") || state.(string) != stateFromCookie {
				return errors.New("Invalid login state")
			}
			token, err := singleton.GetOauth2Config().Exchange(c.Context(), c.Query("code"))
			if err != nil {
				return err
			}

			httpClient := singleton.GetOauth2Config().Client(c.Context(), token)
			client := githubv4.NewClient(httpClient)

			var loginUser model.QueryLoginUser
			if err := client.Query(c.Context(), &loginUser, nil); err != nil {
				return err
			}

			userId, err := loginUser.GetId()
			if err != nil {
				return err
			}

			var user model.User
			user.ID = userId
			if err := singleton.GetDB().FirstOrCreate(&user).Error; err != nil {
				return err
			}

			sid, err := util.GenerateSid(string(loginUser.Viewer.Login))
			if err != nil {
				return err
			}
			user.Sid = sid
			user.GitHubLogin = string(loginUser.Viewer.Login)
			user.SetGitHubToken(token)
			if err := singleton.GetDB().Save(&user).Error; err != nil {
				return err
			}

			c.Cookie(&fiber.Cookie{
				Name:     model.KeyAuthCookie,
				Value:    sid,
				Expires:  time.Now().Add(time.Hour * 24 * 10),
				Secure:   false,
				HTTPOnly: false,
			})
			return c.Render("redirect", map[string]interface{}{
				"URL": "/",
			})
		})
	}

	app.Listen(":80")
}

func userAuthorize(c *fiber.Ctx) error {
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
