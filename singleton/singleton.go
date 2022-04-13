package singleton

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"sync"
	"time"

	"golang.org/x/oauth2"
	GitHubOauth2 "golang.org/x/oauth2/github"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/gofiber/fiber/v2"
	"github.com/naiba/gantt-viewer-for-github-project/model"
	"github.com/patrickmn/go-cache"
)

var (
	db     *gorm.DB
	dbInit = new(sync.Once)

	conf     *model.Config
	confInit = new(sync.Once)

	oauth2config *oauth2.Config
	oauth2init   = new(sync.Once)

	Cache *cache.Cache
)

func init() {
	Cache = cache.New(5*time.Minute, 10*time.Minute)
}

func setupDB() {
	log.Println("connecting to database...")
	var err error
	db, err = gorm.Open(sqlite.Open("data/gantt.db"), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("failed to connect database %+v", err))
	}
	db.AutoMigrate(&model.User{})
}

func GetDB() *gorm.DB {
	dbInit.Do(setupDB)
	return db
}

func readConfig() {
	log.Println("reading config...")
	content, err := ioutil.ReadFile("data/config.json")
	if err != nil {
		panic(fmt.Sprintf("failed to read config file %+v", err))
	}
	conf = &model.Config{}
	err = json.Unmarshal(content, conf)
	if err != nil {
		panic(fmt.Sprintf("failed to parse config file %+v", err))
	}
}

func GetConfig() *model.Config {
	confInit.Do(readConfig)
	return conf
}

func setupOauth2() {
	log.Println("reading oauth2 config...")
	oauth2config = &oauth2.Config{
		ClientID:     GetConfig().GitHubOauthClientID,
		ClientSecret: GetConfig().GitHubOauthClientSecret,
		Scopes:       []string{"user", "repo", "admin:org"},
		Endpoint:     GitHubOauth2.Endpoint,
	}
}

func GetOauth2Config() *oauth2.Config {
	oauth2init.Do(setupOauth2)
	return oauth2config
}

func Map(data fiber.Map) fiber.Map {
	data["Site"] = fiber.Map{
		"Title": "Gantt Viewer for GitHub Project",
		"Brand": "Gantt Viewer",
	}
	return data
}
