package model

import "strconv"

const (
	_ = iota
	UserProject
	OrganizationProject
)

type Project struct {
	Number int
	Title  string
	Public bool
	Owner  string
	Type   uint8

	StartField        bool
	EndField          bool
	ProgressField     bool
	DependenciesField bool
}

func (p *Project) GetURL() string {
	if p.Type == UserProject {
		return "/users/" + p.Owner + "/projects/" + strconv.Itoa(p.Number)
	}
	return "/orgs/" + p.Owner + "/projects/" + strconv.Itoa(p.Number)
}

func (p *Project) IsActionRequired() bool {
	return !(p.DependenciesField && p.ProgressField && p.StartField && p.EndField)
}
