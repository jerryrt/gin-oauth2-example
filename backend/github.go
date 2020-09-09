package backend

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

var github_config *oauth2.Config

type githubUser struct {
	Login   string `json:"login"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Company string `json:"company"`
	URL     string `json:"url"`
}

func getGithubOauthURL() (*oauth2.Config, string) {
	options := CreateClientOptions("github", "https://ginoauth-example.herokuapp.com/callback/github")

	github_config = &oauth2.Config{
		ClientID:     options.getID(),
		ClientSecret: options.getSecret(),
		RedirectURL:  options.getRedirectURL(),
		Scopes: []string{
			"user",
			"repo",
		},
		Endpoint: github.Endpoint,
	}

	state := GenerateState()
	return github_config, state
}

func GithubOauthLogin(ctx *gin.Context) {
	config, state := getGithubOauthURL()
	redirectURL := config.AuthCodeURL(state)

	session := sessions.Default(ctx)
	session.Set("state", state)
	err := session.Save()
	if err != nil {
		_ = ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	ctx.Redirect(http.StatusSeeOther, redirectURL)
}

func GithubCallBack(ctx *gin.Context) {
	session := sessions.Default(ctx)
	state := session.Get("state")
	if state != ctx.Query("state") {
		_ = ctx.AbortWithError(http.StatusUnauthorized, StateError)
		return
	}

	code := ctx.Query("code")
	token, err := github_config.Exchange(ctx, code)
	if err != nil {
		_ = ctx.AbortWithError(http.StatusUnauthorized, err)
		return
	}

	client := github_config.Client(context.TODO(), token)
	userInfo, err := client.Get("https://api.github.com/user")
	if err != nil {
		_ = ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}
	defer userInfo.Body.Close()

	info, err := ioutil.ReadAll(userInfo.Body)
	if err != nil {
		_ = ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	var user githubUser
	__debug__printJSON(info)
	err = json.Unmarshal(info, &user)
	if err != nil {
		_ = ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// redirect to islogin page, and add email, name into url's query string.
	redirectURL, err := url.Parse(IsLoginURL)
	if err != nil {
		_ = ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	query, err := url.ParseQuery(redirectURL.RawQuery)
	if err != nil {
		_ = ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	query.Add("email", user.Email)
	query.Add("name", user.Name)
	query.Add("source", "github")
	redirectURL.RawQuery = query.Encode()

	// 跳轉登入成功畫面(顯示登入資訊)
	ctx.Redirect(http.StatusSeeOther, redirectURL.String())
}
