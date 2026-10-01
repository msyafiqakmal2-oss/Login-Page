package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// Hasher adalah klien untuk layanan Rust (argon2 + audit log).
// Bila HASHER_URL kosong, dipakai fallback sederhana KHUSUS pengembangan.
type Hasher struct {
	url string
	c   *http.Client
}

func NewHasher(url string) *Hasher {
	return &Hasher{url: strings.TrimRight(url, "/"), c: &http.Client{Timeout: 5 * time.Second}}
}

func (h *Hasher) post(path string, in, out any) error {
	b, _ := json.Marshal(in)
	res, err := h.c.Post(h.url+path, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if out == nil {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(out)
}

func (h *Hasher) Hash(pw string) (string, error) {
	if h.url == "" {
		salt := make([]byte, 8)
		rand.Read(salt)
		s := hex.EncodeToString(salt)
		sum := sha256.Sum256([]byte(s + pw))
		return "dev$" + s + "$" + hex.EncodeToString(sum[:]), nil
	}
	var out struct {
		Hash string `json:"hash"`
	}
	err := h.post("/hash", map[string]string{"password": pw}, &out)
	if err == nil && out.Hash == "" {
		err = http.ErrNotSupported
	}
	return out.Hash, err
}

func (h *Hasher) Verify(pw, hash string) bool {
	if h.url == "" {
		p := strings.Split(hash, "$")
		if len(p) != 3 {
			return false
		}
		sum := sha256.Sum256([]byte(p[1] + pw))
		return hex.EncodeToString(sum[:]) == p[2]
	}
	var out struct {
		OK bool `json:"ok"`
	}
	if err := h.post("/verify", map[string]string{"password": pw, "hash": hash}, &out); err != nil {
		return false
	}
	return out.OK
}

// Audit dikirim asinkron agar tidak memperlambat request.
func (h *Hasher) Audit(actor, action, detail string) {
	if h.url == "" {
		return
	}
	go h.post("/audit", map[string]string{"actor": actor, "action": action, "detail": detail}, nil)
}

func (h *Hasher) AuditList() (json.RawMessage, error) {
	if h.url == "" {
		return json.RawMessage("[]"), nil
	}
	res, err := h.c.Get(h.url + "/audit")
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var raw json.RawMessage
	err = json.NewDecoder(res.Body).Decode(&raw)
	return raw, err
}
