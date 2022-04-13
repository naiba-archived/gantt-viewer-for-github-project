package main

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
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
	app.Static("/static", "./static")

	app.Get("/", userAuthorize, func(c *fiber.Ctx) error {
		user := c.Locals(model.KeyAuthorizedUser)
		return c.Render("index", singleton.Map(fiber.Map{
			"User": user,
		}))
	})

	app.Get("/:type/:user/projects/:id/views/:vid", redirectToMainProjectPage)
	app.Get("/:type/:user/projects/:id/", func(c *fiber.Ctx) error {
		// TODO 加载 projects 中的内容
		return c.SendString("bingo")
	})

	{
		app.Get("/dashboard", userAuthorize, loginRequired, func(c *fiber.Ctx) error {
			user := c.Locals(model.KeyAuthorizedUser).(*model.User)
			token, err := user.GetGitHubToken()
			if err != nil {
				return err
			}

			httpClient := singleton.GetOauth2Config().Client(c.Context(), token)
			client := githubv4.NewClient(httpClient)

			var wg sync.WaitGroup
			wg.Add(3)

			var errMix error

			var organizationProjects model.QueryOrganizationProjectsNext
			var repositoriesProjects model.QueryRepositoriesProjectsNext
			var viewerProjectsNext model.QueryViewerProjectsNext

			go func() {
				defer wg.Done()
				if err := client.Query(c.Context(), &organizationProjects, nil); err != nil {
					errMix = err
				}
			}()
			go func() {
				defer wg.Done()
				if err := client.Query(c.Context(), &repositoriesProjects, nil); err != nil {
					errMix = err
				}
			}()
			go func() {
				defer wg.Done()
				if err := client.Query(c.Context(), &viewerProjectsNext, nil); err != nil {
					errMix = err
				}
			}()

			wg.Wait()
			if errMix != nil {
				return errMix
			}

			var projects []model.Project
			projects = append(projects, viewerProjectsNext.Viewer.ProjectsNext.ToProjects()...)
			for _, v := range organizationProjects.Viewer.Organizations.Nodes {
				projects = append(projects, v.ProjectsNext.ToProjects()...)
			}
			for _, v := range repositoriesProjects.Viewer.Repositories.Nodes {
				projects = append(projects, v.ProjectsNext.ToProjects()...)
			}

			log.Printf("%+v", viewerProjectsNext)

			return c.Render("dashboard", singleton.Map(fiber.Map{
				"User":     user,
				"Projects": projects,
			}))
		})

		app.Post("/logout", userAuthorize, loginRequired, func(c *fiber.Ctx) error {
			user := c.Locals(model.KeyAuthorizedUser).(*model.User)
			sid, err := util.GenerateSid(string(user.GitHubLogin))
			if err != nil {
				return err
			}
			user.Sid = sid
			if err := singleton.GetDB().Save(&user).Error; err != nil {
				return err
			}
			c.Cookie(&fiber.Cookie{
				Name:     model.KeyAuthCookie,
				Value:    "",
				Expires:  time.Now(),
				Secure:   false,
				HTTPOnly: false,
			})
			return c.Render("redirect", singleton.Map(fiber.Map{
				"URL": "/",
			}))
		})
	}

	oauth2group := app.Group("/oauth2").Use(userAuthorize).Use(anonymousRequired)
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
			return c.Render("redirect", singleton.Map(fiber.Map{
				"URL": singleton.GetOauth2Config().AuthCodeURL(state, oauth2.AccessTypeOnline),
			}))
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

			var viewer model.QueryLoginUser
			if err := client.Query(c.Context(), &viewer, nil); err != nil {
				return err
			}

			userId, err := viewer.GetId()
			if err != nil {
				return err
			}

			var user model.User
			user.ID = userId
			if err := singleton.GetDB().FirstOrCreate(&user).Error; err != nil {
				return err
			}

			sid, err := util.GenerateSid(string(viewer.Viewer.Login))
			if err != nil {
				return err
			}
			user.Sid = sid
			user.GitHubLogin = string(viewer.Viewer.Login)
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
			return c.Render("redirect", singleton.Map(fiber.Map{
				"URL": "/",
			}))
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

func loginRequired(c *fiber.Ctx) error {
	user := c.Locals(model.KeyAuthorizedUser)
	if user == nil {
		return errors.New("Login required")
	}
	return c.Next()
}

func anonymousRequired(c *fiber.Ctx) error {
	user := c.Locals(model.KeyAuthorizedUser)
	if user != nil {
		return errors.New("Already login")
	}
	return c.Next()
}

func redirectToMainProjectPage(c *fiber.Ctx) error {
	return c.Redirect("/" + c.Params("type") + "/" + c.Params("user") + "/projects/" + c.Params("id"))
}
