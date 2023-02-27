package auth_token

import (
	"log"
	"sync"
	"time"
)

type tokenMaker interface {
	CreateToken(id int64, name string, duration time.Duration) (string, error)
	VerifyToken(token string) (*Payload, error)
}

var once sync.Once
var instance tokenMaker

func TokenMaker() tokenMaker {
	once.Do(func() {
		if instance == nil {
			var err error
			instance, err = NewPasetoMaker()
			if err != nil {
				log.Fatal("failed to load token maker: ", err)
			}
		}
	})

	return instance
}
