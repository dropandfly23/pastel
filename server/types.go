package server

import gitlab "github.com/xanzy/go-gitlab"

type token struct {
	AccessToken string `json:"access_token"`
}

type environment struct {
	Name string
	User *gitlab.User
	Reason string
}
