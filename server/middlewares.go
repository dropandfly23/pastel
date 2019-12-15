package server

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	"github.com/sirupsen/logrus"
	gitlab "github.com/xanzy/go-gitlab"
)

func environmentMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		name := c.Param("name")

		lock.RLock()
		found := false
		for _, e := range envs {
			if e.Name == name {
				found = true
				break
			}
		}
		lock.RUnlock()

		if !found {
			return c.HTML(http.StatusNotFound, "")
		}

		return next(c)
	}
}

func authMiddleware(gitlabURI string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			t, err := c.Cookie("token")
			if err != nil {
				return c.Redirect(http.StatusTemporaryRedirect, "/login")
			}

			git := gitlab.NewOAuthClient(nil, t.Value)
			git.SetBaseURL(fmt.Sprintf("%s/api/v4", gitlabURI))
			me, res, err := git.Users.CurrentUser()
			if err != nil {
				logrus.Warnf("Can't get user with given token %d", res.StatusCode)
				return c.Redirect(http.StatusTemporaryRedirect, "/login")
			}

			c.Set("me", me)

			return next(c)
		}
	}
}
