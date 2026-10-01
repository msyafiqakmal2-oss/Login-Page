package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

type Claims struct {
	Sub  int    `json:"sub"`
	Name string `json:"usr"`
	Role string `json:"role"`
	Exp  int64  `json:"exp"`
}

type authed func(http.ResponseWriter, *http.Request, *Claims)

var b64 = base64.RawURLEncoding

func sign(data string, secret []byte) string {
	m := hmac.New(sha256.New, secret)
	m.Write([]byte(data))
	return b64.EncodeToString(m.Sum(nil))
}

// IssueToken membuat JWT HS256 tanpa dependensi eksternal.
func IssueToken(u *User, secret []byte, ttl time.Duration) string {
	head := b64.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	body, _ := json.Marshal(Claims{u.ID, u.Username, u.Role, time.Now().Add(ttl).Unix()})
	data := head + "." + b64.EncodeToString(body)
	return data + "." + sign(data, secret)
}

func ParseToken(tok string, secret []byte) (*Claims, error) {
	p := strings.Split(tok, ".")
	if len(p) != 3 {
		return nil, errors.New("token rusak")
	}
	if !hmac.Equal([]byte(sign(p[0]+"."+p[1], secret)), []byte(p[2])) {
		return nil, errors.New("tanda tangan salah")
	}
	raw, err := b64.DecodeString(p[1])
	if err != nil {
		return nil, err
	}
	var c Claims
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	if time.Now().Unix() > c.Exp {
		return nil, errors.New("token kedaluwarsa")
	}
	return &c, nil
}

// auth: middleware yang mewajibkan header "Authorization: Bearer <token>".
func (a *App) auth(h authed) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		c, err := ParseToken(tok, a.secret)
		if err != nil {
			fail(w, 401, "Sesi tidak valid. Silakan masuk lagi.")
			return
		}
		h(w, r, c)
	}
}

// admin: pembatas akses khusus role admin.
func (a *App) admin(h authed) authed {
	return func(w http.ResponseWriter, r *http.Request, c *Claims) {
		if c.Role != "admin" {
			fail(w, 403, "Hanya admin yang boleh melakukan ini.")
			return
		}
		h(w, r, c)
	}
}
