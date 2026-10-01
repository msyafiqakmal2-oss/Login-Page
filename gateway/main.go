package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type App struct {
	store  *Store
	hasher *Hasher
	secret []byte
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func respond(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, code int, msg string) {
	respond(w, code, map[string]string{"error": msg})
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		fail(w, 400, "Format data tidak valid.")
		return false
	}
	return true
}

type creds struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (a *App) register(w http.ResponseWriter, r *http.Request) {
	var c creds
	if !decode(w, r, &c) {
		return
	}
	c.Username = strings.TrimSpace(c.Username)
	if len(c.Username) < 3 || len(c.Username) > 32 {
		fail(w, 400, "Nama pengguna harus 3-32 karakter.")
		return
	}
	if len(c.Password) < 8 {
		fail(w, 400, "Kata sandi minimal 8 karakter.")
		return
	}
	hash, err := a.hasher.Hash(c.Password)
	if err != nil {
		fail(w, 502, "Layanan keamanan (Rust) tidak dapat dihubungi.")
		return
	}
	u, err := a.store.CreateUser(c.Username, hash)
	if err != nil {
		fail(w, 409, "Nama pengguna sudah dipakai.")
		return
	}
	a.hasher.Audit(u.Username, "register", "role="+u.Role)
	respond(w, 201, map[string]any{"token": IssueToken(u, a.secret, 12*time.Hour), "user": u})
}

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	var c creds
	if !decode(w, r, &c) {
		return
	}
	u, err := a.store.UserByName(strings.TrimSpace(c.Username))
	if err != nil || !a.hasher.Verify(c.Password, u.Hash) {
		a.hasher.Audit(c.Username, "login_failed", r.RemoteAddr)
		fail(w, 401, "Nama pengguna atau kata sandi salah.")
		return
	}
	a.hasher.Audit(u.Username, "login", r.RemoteAddr)
	respond(w, 200, map[string]any{"token": IssueToken(u, a.secret, 12*time.Hour), "user": u})
}

func (a *App) me(w http.ResponseWriter, r *http.Request, c *Claims) {
	respond(w, 200, c)
}

func (a *App) listNotes(w http.ResponseWriter, r *http.Request, c *Claims) {
	respond(w, 200, a.store.ListNotes(c.Sub))
}

func (a *App) createNote(w http.ResponseWriter, r *http.Request, c *Claims) {
	var in struct{ Title, Body string }
	if !decode(w, r, &in) {
		return
	}
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" || len(in.Title) > 120 {
		fail(w, 400, "Judul wajib diisi (maks. 120 karakter).")
		return
	}
	n := a.store.CreateNote(c.Sub, in.Title, in.Body)
	a.hasher.Audit(c.Name, "note_create", strconv.Itoa(n.ID))
	respond(w, 201, n)
}

func (a *App) deleteNote(w http.ResponseWriter, r *http.Request, c *Claims) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	switch err := a.store.DeleteNote(id, c.Sub, c.Role == "admin"); err {
	case nil:
		a.hasher.Audit(c.Name, "note_delete", strconv.Itoa(id))
		w.WriteHeader(204)
	case ErrForbidden:
		fail(w, 403, "Catatan ini bukan milik Anda.")
	default:
		fail(w, 404, "Catatan tidak ditemukan.")
	}
}

func (a *App) listUsers(w http.ResponseWriter, r *http.Request, c *Claims) {
	respond(w, 200, a.store.ListUsers())
}

func (a *App) deleteUser(w http.ResponseWriter, r *http.Request, c *Claims) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	if id == c.Sub {
		fail(w, 400, "Anda tidak bisa menghapus akun sendiri.")
		return
	}
	u, err := a.store.DeleteUser(id)
	if err != nil {
		fail(w, 404, "Pengguna tidak ditemukan.")
		return
	}
	a.hasher.Audit(c.Name, "user_delete", u.Username)
	w.WriteHeader(204)
}

func (a *App) audit(w http.ResponseWriter, r *http.Request, c *Claims) {
	raw, err := a.hasher.AuditList()
	if err != nil {
		fail(w, 502, "Log audit tidak tersedia.")
		return
	}
	respond(w, 200, raw)
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(t).Round(time.Microsecond))
	})
}

func main() {
	a := &App{
		store:  NewStore(),
		hasher: NewHasher(os.Getenv("HASHER_URL")),
		secret: []byte(getenv("JWT_SECRET", "dev-secret-ganti-di-produksi")),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auth/register", a.register)
	mux.HandleFunc("POST /api/auth/login", a.login)
	mux.HandleFunc("GET /api/me", a.auth(a.me))
	mux.HandleFunc("GET /api/notes", a.auth(a.listNotes))
	mux.HandleFunc("POST /api/notes", a.auth(a.createNote))
	mux.HandleFunc("DELETE /api/notes/{id}", a.auth(a.deleteNote))
	mux.HandleFunc("GET /api/users", a.auth(a.admin(a.listUsers)))
	mux.HandleFunc("DELETE /api/users/{id}", a.auth(a.admin(a.deleteUser)))
	mux.HandleFunc("GET /api/audit", a.auth(a.admin(a.audit)))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		respond(w, 200, map[string]string{"status": "ok"})
	})
	mux.Handle("/", http.FileServer(http.Dir(getenv("WEB_DIR", "../web"))))

	addr := ":" + getenv("PORT", "8080")
	log.Printf("NexusStack gateway (Go) berjalan di %s", addr)
	log.Fatal(http.ListenAndServe(addr, logging(mux)))
}
