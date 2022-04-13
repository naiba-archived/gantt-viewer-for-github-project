package model

import (
	"encoding/json"

	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	GitHubLogin string
	GitHubToken string

	Sid string `gorm:"unique_index"`
}

func (u *User) SetGitHubToken(token *oauth2.Token) error {
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}
	u.GitHubToken = string(data)
	return nil
}

func (u *User) GetGitHubToken() (*oauth2.Token, error) {
	var token oauth2.Token
	return &token, json.Unmarshal([]byte(u.GitHubToken), &token)
}
