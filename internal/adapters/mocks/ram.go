package mocks

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/RobMil91/free-orgx/internal/ports"
)

var _ ports.UserRepo = (*RAM)(nil)

const (
	// CookieLiveTime = time.Minute * 10
	CookieLiveTime = time.Minute * 1
)

type RAM struct {
	Users map[string]ports.User
}

func (r *RAM) GetUserToken(ctx context.Context, name, password string) (*ports.SessionCookie, error) {
	u, ok := r.Users[name]

	if !ok {
		return nil, fmt.Errorf("%s %w", name, ports.UserNotFound)
	}

	if u.Password != password {
		return nil, errors.New("bad password")
	}

	newCookie, err := CreateRandStr(32)
	if err != nil {
		return nil, fmt.Errorf("could not create cookie %w", err)
	}

	u.Cookie.CreateTime = time.Now()
	u.Cookie.Value = *newCookie
	r.Users[name] = u

	return &u.Cookie, nil
}

// this should not be adapter logic refacotor to use in controller
func (r *RAM) IsValid(ctx context.Context, cookie string) (*ports.User, error) {
	for _, u := range r.Users {

		expirationTime := u.Cookie.CreateTime.Add(CookieLiveTime)
		if u.Cookie.Value == cookie && time.Now().Before(expirationTime) {
			return &u, nil
		}
	}

	return nil, errors.New("no user with this active token")
}

// TODO: this probably bad random access, would need refactor for sufisticated solution. Do not use with sensitive data.
func CreateRandStr(length int) (*string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	randStr := base64.URLEncoding.EncodeToString(b)
	return &randStr, nil
}
