package util

import (
	"sync"

	"github.com/slack-go/slack"
	"github.com/yeom-c/data-api/app"
)

type slackInstance struct {
	Client *slack.Client
}

var once sync.Once
var instance *slackInstance

func Slack() *slackInstance {
	once.Do(func() {
		if instance == nil {
			instance = &slackInstance{
				Client: slack.New(app.Config().SlackBotOauthToken),
			}
		}
	})

	return instance
}
