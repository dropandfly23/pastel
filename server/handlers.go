package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/labstack/echo"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	gitlab "github.com/xanzy/go-gitlab"
)

func loginHandler(URI, gitlabURI, gitlabAppID string) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.Redirect(http.StatusTemporaryRedirect,
			fmt.Sprintf("%s/oauth/authorize?client_id=%s&redirect_uri=%s&response_type=code&state=%s",
				gitlabURI, gitlabAppID, fmt.Sprintf("%s/callback", URI), uuid.NewV4().String()))
	}
}

func environmentsToggleHandler(c echo.Context) error {
	me := c.Get("me").(*gitlab.User)
	name := c.Param("name")

	lock.Lock()
	defer lock.Unlock()

	for _, e := range envs {
		if e.Name == name {
			if e.User == nil {
				e.User = me
			} else {
				e.User = nil
			}
		}
	}

	return c.Redirect(http.StatusTemporaryRedirect, "/")
}

func environmentsUpdateReasonHandler(c echo.Context) error {
	reason := c.FormValue("reason")
	name := c.Param("name")

	lock.Lock()
	defer lock.Unlock()

	for _, e := range envs {
		if e.Name == name {
			e.Reason = reason
		}
	}
	return c.Redirect(http.StatusTemporaryRedirect, "/")
}

func callbackHandler(URI, gitlabURI, gitlabAppID, gitlabSecret string) echo.HandlerFunc {
	return func(c echo.Context) error {
		params := url.Values{}
		params.Add("client_id", gitlabAppID)
		params.Add("client_secret", gitlabSecret)
		params.Add("code", c.QueryParam("code"))
		params.Add("grant_type", "authorization_code")
		params.Add("redirect_uri", fmt.Sprintf("%s/callback", URI))

		res, err := http.PostForm(fmt.Sprintf("%s/oauth/token", gitlabURI), params)
		if err != nil {
			logrus.Warn("Can't get access token from code", err)
			return c.Redirect(http.StatusTemporaryRedirect, "/")
		}

		var body bytes.Buffer
		body.ReadFrom(res.Body)

		var t token
		if err := json.Unmarshal(body.Bytes(), &t); err != nil {
			return c.Redirect(http.StatusTemporaryRedirect, "/")
		}

		c.SetCookie(&http.Cookie{
			Name:  "token",
			Value: t.AccessToken,
		})

		return c.Redirect(http.StatusTemporaryRedirect, "/")
	}
}
