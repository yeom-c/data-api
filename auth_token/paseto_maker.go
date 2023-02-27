package auth_token

import (
	"fmt"
	"time"

	"github.com/vk-rv/pvx"
	"github.com/yeom-c/data-api/app"
	"golang.org/x/crypto/chacha20poly1305"
)

type pasetoMaker struct {
	symmetricKey []byte
}

func NewPasetoMaker() (tokenMaker, error) {
	symmetricKey := app.Config().AuthTokenSymmetricKey
	if len(symmetricKey) < chacha20poly1305.KeySize {
		return nil, fmt.Errorf("secret key is too short")
	}

	maker := &pasetoMaker{
		symmetricKey: []byte(symmetricKey),
	}

	return maker, nil
}

func (maker *pasetoMaker) CreateToken(id int64, name string, duration time.Duration) (string, error) {
	payload, err := NewPayload(id, name, duration)
	if err != nil {
		return "", err
	}

	key := pvx.NewSymmetricKey(maker.symmetricKey, pvx.Version4)
	pv4 := pvx.NewPV4Local()

	return pv4.Encrypt(key, payload)
}

func (maker *pasetoMaker) VerifyToken(token string) (*Payload, error) {
	payload := &Payload{}

	symK := pvx.NewSymmetricKey(maker.symmetricKey, pvx.Version4)
	pv4 := pvx.NewPV4Local()
	err := pv4.Decrypt(token, symK).ScanClaims(payload)
	if err != nil {
		return nil, err
	}

	return payload, nil
}
