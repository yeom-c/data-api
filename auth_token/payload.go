package auth_token

import (
	"fmt"
	"time"
)

type Payload struct {
	Id        int64     `json:"id"`
	Name      string    `json:"name"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiredAt time.Time `json:"expired_at"`
}

func NewPayload(id int64, name string, duration time.Duration) (*Payload, error) {
	payload := &Payload{
		Id:        id,
		Name:      name,
		IssuedAt:  time.Now(),
		ExpiredAt: time.Now().Add(duration),
	}

	return payload, nil
}

func (p *Payload) Valid() error {
	if time.Now().After(p.ExpiredAt) {
		return fmt.Errorf("인증 토큰 만료")
	}

	return nil
}
