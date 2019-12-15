package server

import (
	"html/template"
	"io"
	"net/http"

	"github.com/labstack/echo"
)

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

type page struct {
	Title string
}

type indexPage struct {
	page
	Environments []*environment
}

func homePage(c echo.Context) error {
	lock.RLock()
	defer lock.RUnlock()
	return c.Render(http.StatusOK, "index", indexPage{
		page:         page{Title: "Pastel - Home"},
		Environments: envs,
	})
}

func loginPage(c echo.Context) error {
	return c.Render(http.StatusOK, "login", page{Title: "Pastel - Login"})
}
