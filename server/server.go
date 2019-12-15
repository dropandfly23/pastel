package server

import (
	"fmt"
	"html/template"
	"sync"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

var lock sync.RWMutex
var envs []*environment

// Start the home server.
func Start(port int, URI string, listEnvs []string, gitlabAppID, gitlabSecret, gitlabURI string) error {
	lock.Lock()
	for _, e := range listEnvs {
		envs = append(envs, &environment{Name: e})
	}
	lock.Unlock()

	e := echo.New()

	t := &Template{templates: template.Must(template.ParseGlob("public/templates/*.html"))}
	e.Renderer = t

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/callback", callbackHandler(URI, gitlabURI, gitlabAppID, gitlabSecret))
	e.GET("/login", loginPage)

	e.Any("/", homePage, authMiddleware(gitlabURI))
	api := e.Group("/api")
	api.GET("/login", loginHandler(URI, gitlabURI, gitlabAppID))
	api.POST("/environments/:name/toggle", environmentsToggleHandler, authMiddleware(gitlabURI), environmentMiddleware)
	api.POST("/environments/:name/update_reason", environmentsUpdateReasonHandler, authMiddleware(gitlabURI), environmentMiddleware)

	return e.Start(fmt.Sprintf(":%d", port))
}
