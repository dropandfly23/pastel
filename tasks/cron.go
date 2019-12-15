package tasks

import (
	"os"
	"os/signal"
	"time"

	"github.com/sirupsen/logrus"
	cron "gopkg.in/robfig/cron.v2"
)

// CronWrapper schedule task with cron if enabled.
func CronWrapper(enable bool, period string, job func() error) error {
	if !enable {
		return job()
	}

	task := func() {
		if err := job(); err != nil {
			logrus.Error(err)
		}
	}
	go task()

	c := cron.New()
	c.AddFunc(period, task)
	c.Start()

	logrus.Info("Task running with cron")

	t := time.NewTicker(time.Second)

	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt)

	for {
		select {
		case <-t.C:
			continue
		case <-ch:
			logrus.Info("Task stopped")
			os.Exit(0)
		}
	}
}
