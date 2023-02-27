package util

import (
	"context"
	"errors"
	"io"

	"github.com/goccy/go-json"
	"github.com/yeom-c/data-api/app"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const GoogleUserInfoApi = "https://www.googleapis.com/oauth2/v3/userinfo"

type GUser struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	AuthProvider string
}

func GetGoogleUserInfo(authCode string) (authUser GUser, err error) {
	oAuthConf := &oauth2.Config{
		ClientID:     app.Config().AuthGoogleClientId,
		ClientSecret: app.Config().AuthGoogleClientSecret,
		RedirectURL:  "postmessage",
		Endpoint:     google.Endpoint,
	}

	token, err := oAuthConf.Exchange(context.Background(), authCode)
	if err != nil {
		err = errors.New("인증 실패")
		return
	}

	client := oAuthConf.Client(context.Background(), token)
	response, err := client.Get(GoogleUserInfoApi)
	if err != nil {
		err = errors.New("유저정보 확인 불가")
		return
	}
	defer response.Body.Close()

	userInfo, err := io.ReadAll(response.Body)
	if err != nil {
		err = errors.New("유저정보 확인 불가")
		return
	}

	json.Unmarshal(userInfo, &authUser)
	authUser.AuthProvider = "google"

	return
}
