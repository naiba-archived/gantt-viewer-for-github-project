package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/html"
	"github.com/patrickmn/go-cache"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"

	"github.com/naiba/gantt-viewer-for-github-project/model"
	"github.com/naiba/gantt-viewer-for-github-project/router"
	"github.com/naiba/gantt-viewer-for-github-project/singleton"
	"github.com/naiba/gantt-viewer-for-github-project/util"
)

func main() {
	engine := html.New("./views", ".html")
	engine.AddFunc("json", func(x interface{}) string {
		data, _ := json.Marshal(x)
		return string(data)
	})

	if singleton.GetConfig().Debug {
		engine.Reload(true)
		engine.Debug(true)
	}

	app := fiber.New(
		fiber.Config{
			Views: engine,
		},
	)
	app.Use(recover.New())
	app.Static("/static", "./static")

	app.Get("/", router.UserAuthorize, func(c *fiber.Ctx) error {
		return c.Render("index", singleton.Map(c, fiber.Map{}))
	})

	app.Get("/:type/:users/projects/:id/views/:vid", router.RedirectToMainProjectPage)
	app.Get("/:type/:users/projects/:id/", router.UserAuthorize, router.LoginRequired, func(c *fiber.Ctx) error {
		id, err := c.ParamsInt("id")
		if err != nil {
			return err
		}
		if c.Params("type") == "orgs" {
			return router.ProjectHome[model.QueryOrganizationProjectNext](c, c.Params("users"), id)
		}
		if c.Params("type") == "users" {
			return router.ProjectHome[model.QueryUserProjectNext](c, c.Params("users"), id)
		}
		return errors.New("Invalid type")
	})

	{
		app.Get("/dashboard", router.UserAuthorize, router.LoginRequired, func(c *fiber.Ctx) error {
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

			projectLoadVariables := map[string]interface {
			}{
				"projectFieldFirst":       githubv4.Int(100),
				"projectFieldValuesFirst": githubv4.Int(0),
				"projectItemsFirst":       githubv4.Int(0),
				"projectItemsAfter":       (*githubv4.String)(nil),
			}

			go func() {
				defer wg.Done()
				if err := client.Query(c.Context(), &organizationProjects, projectLoadVariables); err != nil {
					errMix = err
				}
			}()
			go func() {
				defer wg.Done()
				if err := client.Query(c.Context(), &repositoriesProjects, projectLoadVariables); err != nil {
					errMix = err
				}
			}()
			go func() {
				defer wg.Done()
				if err := client.Query(c.Context(), &viewerProjectsNext, projectLoadVariables); err != nil {
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

			return c.Render("dashboard", singleton.Map(c, fiber.Map{
				"Projects": projects,
			}))
		})

		app.Post("/logout", router.UserAuthorize, router.LoginRequired, func(c *fiber.Ctx) error {
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
			return c.Render("redirect", singleton.Map(c, fiber.Map{
				"URL": "/",
			}))
		})
	}

	oauth2group := app.Group("/oauth2").Use(router.UserAuthorize).Use(router.AnonymousRequired)
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
			return c.Render("redirect", singleton.Map(c, fiber.Map{
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
			return c.Render("redirect", singleton.Map(c, fiber.Map{
				"URL": "/",
			}))
		})
	}

	app.Listen(":80")
}
