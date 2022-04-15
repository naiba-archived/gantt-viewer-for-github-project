package model

import (
	"errors"
	"fmt"
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

type GitHubIssueLikeAssignees struct {
	Nodes []struct {
		Login     githubv4.String
		AvatarUrl githubv4.String
	}
}

type GitHubIssueLikeItem struct {
	Url       githubv4.String
	Milestone struct {
		Title githubv4.String
	}
	Assignees GitHubIssueLikeAssignees `graphql:"assignees(first: 100)"`
}

type GitHubDraftIssue struct {
	Assignees GitHubIssueLikeAssignees `graphql:"assignees(first: 100)"`
}

type GitHubProjectNext struct {
	Owner           GitHubProjectsNextOwner
	Number          githubv4.Int
	Public          githubv4.Boolean
	Title           githubv4.String
	Url             githubv4.String
	ViewerCanUpdate githubv4.Boolean
	Fields          struct {
		Nodes []struct {
			ID         githubv4.ID
			Name       githubv4.String
			DatabaseId githubv4.Int
			DataType   githubv4.String
			Settings   githubv4.String
		}
	} `graphql:"fields(first: $projectFieldFirst)"`
	Items GitHubProjectNextItem `graphql:"items(first: $projectItemsFirst, after: $projectItemsAfter)"`
}

type GitHubProjectNextItem struct {
	Nodes []struct {
		FieldValues struct {
			Nodes []struct {
				Value        githubv4.String
				ProjectField struct {
					Name githubv4.String
				}
			}
		} `graphql:"fieldValues(first: $projectFieldValuesFirst)"`
		Content struct {
			DraftIssue  GitHubDraftIssue    `graphql:"... on DraftIssue"`
			Issue       GitHubIssueLikeItem `graphql:"... on Issue"`
			PullRequest GitHubIssueLikeItem `graphql:"... on PullRequest"`
		}
	}
	PageInfo struct {
		HasNextPage githubv4.Boolean
		EndCursor   githubv4.String
	}
}

var getGitHubIssueMatch = regexp.MustCompile(`https:\/\/github\.com\/([^\/]*)\/([^\/]*)\/[^\/]*\/(\d*)`)

func (pi GitHubProjectNextItem) ToGantts() []Gantt {
	var gantts []Gantt
	for i := 0; i < len(pi.Nodes); i++ {
		var gantt Gantt
		if pi.Nodes[i].Content.Issue.Milestone.Title != "" {
			gantt.Milestone = string(pi.Nodes[i].Content.Issue.Milestone.Title)
		} else if pi.Nodes[i].Content.PullRequest.Milestone.Title != "" {
			gantt.Milestone = string(pi.Nodes[i].Content.PullRequest.Milestone.Title)
		}
		if pi.Nodes[i].Content.Issue.Url != "" {
			matches := getGitHubIssueMatch.FindStringSubmatch(string(pi.Nodes[i].Content.Issue.Url))
			gantt.Id = fmt.Sprintf("%s/%s#%s", matches[1], matches[2], matches[3])
		}
		if gantt.Id == "" && pi.Nodes[i].Content.PullRequest.Url != "" {
			matches := getGitHubIssueMatch.FindStringSubmatch(string(pi.Nodes[i].Content.PullRequest.Url))
			gantt.Id = fmt.Sprintf("%s/%s#%s", matches[1], matches[2], matches[3])
		}
		for j := 0; j < len(pi.Nodes[i].Content.Issue.Assignees.Nodes); j++ {
			assignee := pi.Nodes[i].Content.Issue.Assignees.Nodes[j]
			gantt.Assignees = append(gantt.Assignees, struct {
				AvatarUrl string "json:\"avatar_url,omitempty\""
				Login     string "json:\"login,omitempty\""
			}{
				AvatarUrl: string(assignee.AvatarUrl),
				Login:     string(assignee.Login),
			})
		}
		for j := 0; j < len(pi.Nodes[i].FieldValues.Nodes); j++ {
			fieldVal := pi.Nodes[i].FieldValues.Nodes[j]
			switch fieldVal.ProjectField.Name {
			case "Title":
				if gantt.Id == "" {
					gantt.Id = string(fieldVal.Value)
				}
				gantt.Name = string(fieldVal.Value)
			case "Start":
				gantt.Start = strings.Split(string(fieldVal.Value), "T")[0]
			case "End":
				gantt.End = strings.Split(string(fieldVal.Value), "T")[0]
			case "Progress":
				gantt.Progress, _ = strconv.Atoi(string(fieldVal.Value))
			case "Dependencies":
				gantt.Dependencies = string(fieldVal.Value)
			}
		}
		gantts = append(gantts, gantt)
	}
	return gantts
}

type GitHubProjectsNext struct {
	Nodes []GitHubProjectNext
}

type QueryOrganizationProjectsNext struct {
	Viewer struct {
		Organizations struct {
			Nodes []struct {
				ProjectsNext GitHubProjectsNext `graphql:"projectsNext(first: 100, query: \"status:open\")"`
				Login        githubv4.String
			}
		} `graphql:"organizations(first: 40)"`
	}
}

type QueryRepositoriesProjectsNext struct {
	Viewer struct {
		Repositories struct {
			Nodes []struct {
				ProjectsNext GitHubProjectsNext `graphql:"projectsNext(first: 100, query: \"status:open\")"`
			}
		} `graphql:"repositories(first: 40)"`
	}
}

type QueryViewerProjectsNext struct {
	Viewer struct {
		ProjectsNext GitHubProjectsNext `graphql:"projectsNext(first: 100, query: \"status:open\")"`
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
	for _, node := range gpn.Nodes {
		project := Project{
			Number: int(node.Number),
			Title:  string(node.Title),
			Public: bool(node.Public),
			Owner:  string(node.Owner.User.Login),
			Type:   node.GetType(),
		}
		for _, field := range node.Fields.Nodes {
			switch field.Name {
			case "Start":
				project.StartField = field.DataType == "DATE"
			case "End":
				project.EndField = field.DataType == "DATE"
			case "Progress":
				project.ProgressField = field.DataType == "NUMBER"
			case "Dependencies":
				project.DependenciesField = field.DataType == "TEXT"
			}
		}
		projects = append(projects, project)
	}
	return projects
}

type QueryOrganizationProjectNext struct {
	Organization struct {
		ProjectNext GitHubProjectNext `graphql:"projectNext(number: $projectNumber)"`
	} `graphql:"organization(login: $projectOwner)"`
}

func (q QueryOrganizationProjectNext) GetProjectNext() GitHubProjectNext {
	return q.Organization.ProjectNext
}

type QueryUserProjectNext struct {
	User struct {
		ProjectNext GitHubProjectNext `graphql:"projectNext(number: $projectNumber)"`
	} `graphql:"user(login: $projectOwner)"`
}

func (q QueryUserProjectNext) GetProjectNext() GitHubProjectNext {
	return q.User.ProjectNext
}

type GetGitHubProjectExt interface {
	GetProjectNext() GitHubProjectNext
}
