package singleton

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
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

	Cache   *cache.Cache
	Version string
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
	if GetConfig().Debug {
		db = db.Debug()
	}
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

func Map(c *fiber.Ctx, data fiber.Map) fiber.Map {
	user := c.Locals(model.KeyAuthorizedUser)
	theme := c.Cookies("theme")
	if theme == "" {
		theme = "aqua"
	}
	data["Site"] = fiber.Map{
		"Title":   "Gantt Viewer for GitHub Project",
		"Brand":   "Gantt Viewer",
		"User":    user,
		"Theme":   theme,
		"Version": Version,
	}
	return data
}

func GenerateRandomString(n int) (string, error) {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-"
	ret := make([]byte, n)
	for i := 0; i < n; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		ret[i] = letters[num.Int64()]
	}
	return string(ret), nil
}

func GenerateSid(user string) (string, error) {
	var randomBytes = make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	randomBytes = append([]byte(user), randomBytes...)
	return hex.EncodeToString(randomBytes), nil
}
