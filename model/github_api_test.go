package model

import (
	"fmt"
	"testing"

	"github.com/shurcooL/githubv4"
)

func TestQueryLoginUserGetId(t *testing.T) {
	user := QueryLoginUser{}
	var userId uint = 29243953
	user.Viewer.AvatarUrl = githubv4.String(fmt.Sprintf("https://avatars.githubusercontent.com/u/%d?u=blablablabla&v=4", userId))

	id, err := user.GetId()
	if err != nil {
		t.Fatal(err)
	}
	if id != userId {
		t.Fatalf("id = %d, want %d", id, userId)
	}
}
