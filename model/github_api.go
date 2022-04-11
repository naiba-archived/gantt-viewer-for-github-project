package model

import (
	"errors"
	"regexp"
	"strconv"

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
