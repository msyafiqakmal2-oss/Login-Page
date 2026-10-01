package main

import (
	"errors"
	"sort"
	"sync"
	"time"
)

type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	Hash      string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

type Note struct {
	ID        int       `json:"id"`
	OwnerID   int       `json:"owner_id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

var (
	ErrExists    = errors.New("sudah ada")
	ErrNotFound  = errors.New("tidak ditemukan")
	ErrForbidden = errors.New("tidak diizinkan")
)

// Store: penyimpanan in-memory. Ganti dengan Postgres/SQLite dengan
// mengimplementasikan method yang sama (lihat docs/UPGRADE.md).
type Store struct {
	mu    sync.RWMutex
	users map[int]*User
	notes map[int]*Note
	nextU int
	nextN int
}

func NewStore() *Store {
	return &Store{users: map[int]*User{}, notes: map[int]*Note{}, nextU: 1, nextN: 1}
}

func (s *Store) CreateUser(name, hash string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.users {
		if u.Username == name {
			return nil, ErrExists
		}
	}
	role := "user"
	if len(s.users) == 0 {
		role = "admin" // pengguna pertama otomatis admin
	}
	u := &User{ID: s.nextU, Username: name, Role: role, Hash: hash, CreatedAt: time.Now()}
	s.users[u.ID] = u
	s.nextU++
	return u, nil
}

func (s *Store) UserByName(name string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.users {
		if u.Username == name {
			return u, nil
		}
	}
	return nil, ErrNotFound
}

func (s *Store) ListUsers() []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*User, 0, len(s.users))
	for _, u := range s.users {
		out = append(out, u)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Store) DeleteUser(id int) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[id]
	if !ok {
		return nil, ErrNotFound
	}
	delete(s.users, id)
	for nid, n := range s.notes { // hapus catatan milik user
		if n.OwnerID == id {
			delete(s.notes, nid)
		}
	}
	return u, nil
}

func (s *Store) CreateNote(owner int, title, body string) *Note {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := &Note{ID: s.nextN, OwnerID: owner, Title: title, Body: body, CreatedAt: time.Now()}
	s.notes[n.ID] = n
	s.nextN++
	return n
}

func (s *Store) ListNotes(owner int) []*Note {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*Note{}
	for _, n := range s.notes {
		if n.OwnerID == owner {
			out = append(out, n)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out
}

func (s *Store) DeleteNote(id, actor int, isAdmin bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, ok := s.notes[id]
	if !ok {
		return ErrNotFound
	}
	if n.OwnerID != actor && !isAdmin {
		return ErrForbidden
	}
	delete(s.notes, id)
	return nil
}
