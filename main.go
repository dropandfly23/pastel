package main

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"gitlab.forge.orange-labs.fr/indigo/tools/pastel/server"
	"gitlab.forge.orange-labs.fr/indigo/tools/pastel/tasks"
	cli "gopkg.in/urfave/cli.v1"
)

func main() {
	app := cli.NewApp()

	logrus.SetLevel(logrus.DebugLevel)

	gitlabURIFlag := cli.StringFlag{Name: "gitlab.uri", Value: "https://gitlab.forge.orange-labs.fr", EnvVar: "PASTEL_GITLAB_URI"}

	app.Commands = []cli.Command{
		{
			Name:  "start",
			Usage: "Start the Pastel server",
			Flags: []cli.Flag{
				cli.IntFlag{Name: "server.port", Value: 8080, EnvVar: "PASTEL_SERVER_PORT"},
				cli.StringFlag{Name: "server.uri", Value: "http://localhost:8080", EnvVar: "PASTEL_SERVER_URI"},
				cli.StringSliceFlag{Name: "server.env-list", Value: &cli.StringSlice{"0"}, EnvVar: "PASTEL_SERVER_ENV_LIST"},
				cli.StringFlag{Name: "gitlab.app-id", Value: "", EnvVar: "PASTEL_GITLAB_APP_ID"},
				cli.StringFlag{Name: "gitlab.secret", Value: "", EnvVar: "PASTEL_GITLAB_SECRET"},
				gitlabURIFlag,
			},
			Action: func(c *cli.Context) error {
				return server.Start(c.Int("server.port"), c.String("server.uri"), c.StringSlice("server.env-list"),
					c.String("gitlab.app-id"), c.String("gitlab.secret"), c.String("gitlab.uri"))
			},
		}, {
			Name:  "tasks",
			Usage: "Start a Pastel task",
			Flags: []cli.Flag{
				cli.BoolFlag{Name: "tasks.dry-run", EnvVar: "PASTEL_TASKS_DRY_RUN"},
				cli.BoolFlag{Name: "tasks.cron", EnvVar: "PASTEL_TASKS_CRON"},
				cli.StringFlag{Name: "tasks.period", Value: "@daily", EnvVar: "PASTEL_TASKS_PERIOD"},
			},
			Subcommands: []cli.Command{
				{
					Name:  "merge-requests",
					Usage: "Check merge requests branches deletion",
					Flags: []cli.Flag{
						gitlabURIFlag,
						cli.StringFlag{Name: "gitlab.key", Value: "", EnvVar: "PASTEL_GITLAB_KEY"},
						cli.StringFlag{Name: "gitlab.project-id", Value: "", EnvVar: "PASTEL_GITLAB_PROJECT_ID"},
						cli.StringFlag{Name: "tasks.branches-to-exclude", Value: "", EnvVar: "PASTEL_TASKS_BRANCHES_TO_EXCLUDE"},
						cli.StringFlag{Name: "mattermost.webhook-uri", Value: "", EnvVar: "PASTEL_MATTERMOST_WEBHOOK_URI"},
					},
					Action: func(c *cli.Context) error {
						return tasks.CronWrapper(c.GlobalBool("tasks.cron"), c.GlobalString("tasks.period"), func() error {
							return tasks.MergeRequests(c.String("gitlab.uri"),
								c.String("gitlab.key"), c.String("gitlab.project-id"),
								c.String("mattermost.webhook-uri"),
								strings.Split(c.String("tasks.branches-to-exclude"), ","),
								c.GlobalBool("tasks.dry-run"))
						})
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Errorf("%+v", err)
	}
}
