package model

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/shurcooL/githubv4"
)

var matchExactId = regexp.MustCompile(`https:\/\/avatars.githubusercontent.com\/u\/(\d*)\?`)

type QueryLoginUser struct {
	Viewer struct {
		Login     githubv4.String
		AvatarUrl githubv4.String
		CreatedAt githubv4.DateTime
	}
}

func (u QueryLoginUser) GetId() (uint, error) {
	matches := matchExactId.FindAllStringSubmatch(string(u.Viewer.AvatarUrl), -1)
	if len(matches[0]) != 2 {
		return 0, errors.New("Invalid avatar url")
	}
	id, err := strconv.ParseUint(matches[0][1], 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}

type GitHubProjectsNextOwner struct {
	Organization struct {
		Login githubv4.String
	} `graphql:"... on Organization"`
	User struct {
		Login githubv4.String
	} `graphql:"... on User"`
}

type GitHubProjectNext struct {
	Owner           GitHubProjectsNextOwner
	Number          githubv4.Int
	Public          githubv4.Boolean
	Title           githubv4.String
	Url             githubv4.String
	ViewerCanUpdate githubv4.Boolean
	Fields          struct {
		Edges []struct {
			Node struct {
				ID         githubv4.ID
				Name       githubv4.String
				DatabaseId githubv4.Int
				DataType   githubv4.String
				Settings   githubv4.String
			}
		}
	} `graphql:"fields(first: 100)"`
}

type GitHubProjectsNext struct {
	Edges []struct {
		Node GitHubProjectNext
	}
}

type QueryOrganizationProjectsNext struct {
	Viewer struct {
		Organizations struct {
			Nodes []struct {
				ProjectsNext GitHubProjectsNext `graphql:"projectsNext(first: 100)"`
				Login        githubv4.String
			}
		} `graphql:"organizations(first: 40)"`
	}
}

type QueryRepositoriesProjectsNext struct {
	Viewer struct {
		Repositories struct {
			Nodes []struct {
				ProjectsNext GitHubProjectsNext `graphql:"projectsNext(first: 100)"`
			}
		} `graphql:"repositories(first: 40)"`
	}
}

type QueryViewerProjectsNext struct {
	Viewer struct {
		ProjectsNext GitHubProjectsNext `graphql:"projectsNext(first: 100)"`
	}
}

func (gpn *GitHubProjectNext) GetType() uint8 {
	if strings.HasPrefix(string(gpn.Url), "https://github.com/users/") {
		return UserProject
	}
	return OrganizationProject
}

func (gpn GitHubProjectsNext) ToProjects() []Project {
	var projects []Project
	for _, edge := range gpn.Edges {
		project := Project{
			Number: int(edge.Node.Number),
			Title:  string(edge.Node.Title),
			Public: bool(edge.Node.Public),
			Owner:  string(edge.Node.Owner.User.Login),
			Type:   edge.Node.GetType(),
		}
		for _, field := range edge.Node.Fields.Edges {
			switch field.Node.Name {
			case "Start":
				project.StartField = field.Node.DataType == "DATE"
			case "End":
				project.EndField = field.Node.DataType == "DATE"
			case "Progress":
				project.ProgressField = field.Node.DataType == "NUMBER"
			case "Dependencies":
				project.DependenciesField = field.Node.DataType == "TEXT"
			}
		}
		projects = append(projects, project)
	}
	return projects
}
