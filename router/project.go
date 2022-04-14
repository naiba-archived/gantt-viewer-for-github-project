package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/naiba/gantt-viewer-for-github-project/model"
	"github.com/naiba/gantt-viewer-for-github-project/singleton"
	"github.com/shurcooL/githubv4"
)

func RedirectToMainProjectPage(c *fiber.Ctx) error {
	return c.Redirect("/" + c.Params("type") + "/" + c.Params("users") + "/projects/" + c.Params("id"))
}

func ProjectHome[T model.GetGitHubProjectExt](c *fiber.Ctx, owner string, number int) error {
	user := c.Locals(model.KeyAuthorizedUser).(*model.User)
	token, err := user.GetGitHubToken()
	if err != nil {
		return err
	}

	httpClient := singleton.GetOauth2Config().Client(c.Context(), token)
	client := githubv4.NewClient(httpClient)
	projectLoadVariables := map[string]interface {
	}{
		"projectOwner":            githubv4.String(owner),
		"projectNumber":           githubv4.Int(number),
		"projectFieldFirst":       githubv4.Int(0),
		"projectFieldValuesFirst": githubv4.Int(100),
		"projectItemsFirst":       githubv4.Int(100),
		"projectItemsAfter":       (*githubv4.String)(nil),
	}

	var q T
	var qList []T
	if err := client.Query(c.Context(), &q, projectLoadVariables); err != nil {
		return err
	}
	qList = append(qList, q)

	var projectTitle, projectOwner, projectUrl string

	if len(q.GetProjectNext().Items.Nodes) > 0 {
		projectTitle = string(q.GetProjectNext().Title)
		projectOwner = string(q.GetProjectNext().Owner.User.Login)
		projectUrl = string(q.GetProjectNext().Url)
		for q.GetProjectNext().Items.PageInfo.HasNextPage {
			projectLoadVariables["projectItemsAfter"] = q.GetProjectNext().Items.PageInfo.EndCursor
			var q T
			if err := client.Query(c.Context(), &q, projectLoadVariables); err != nil {
				return err
			}
			qList = append(qList, q)
		}
	}

	var gantts []model.Gantt
	for i := 0; i < len(qList); i++ {
		gantts = append(gantts, qList[i].GetProjectNext().Items.ToGantts()...)
	}

	return c.Render("project", singleton.Map(c, fiber.Map{
		"Title":  projectTitle,
		"URL":    projectUrl,
		"Owner":  projectOwner,
		"Gantts": gantts,
	}))
}
