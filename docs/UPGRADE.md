# Peta jalan upgrade NexusStack

Setiap modul berdiri sendiri, jadi bisa ditingkatkan tanpa menyentuh yang lain.

| Modul | Berkas | Upgrade yang disarankan |
|---|---|---|
| Penyimpanan | `gateway/store.go` | Ganti map in-memory dengan PostgreSQL/SQLite. Pertahankan nama method agar handler tidak berubah. |
| Auth | `gateway/auth.go` | Tambah refresh token, cookie HttpOnly, rate limit login. |
| Klien Rust | `gateway/hasher.go` | Ganti HTTP dengan gRPC, tambah retry & circuit breaker. |
| Layanan Rust | `hasher/src/main.rs` | Simpan audit ke disk/DB, tambah endpoint metrik Prometheus. |
| Frontend | `web/` | Pindah ke React/Svelte; API tetap `/api/*`. |

## Ide tambahan lintas bahasa
- **Python** (`analytics/`): laporan penggunaan dari log audit dengan FastAPI.
- **TypeScript**: frontend bertipe dengan Vite, tipe dibuat dari OpenAPI.
- **Node/Deno**: worker notifikasi email.
- **Zig/C++**: modul performa tinggi bila ada kebutuhan khusus.
- **CI**: GitHub Actions untuk `go vet`, `cargo clippy`, dan build Docker.

## Daftar endpoint
| Metode | Path | Akses |
|---|---|---|
| POST | `/api/auth/register` | publik |
| POST | `/api/auth/login` | publik |
| GET | `/api/me` | login |
| GET/POST | `/api/notes` | login |
| DELETE | `/api/notes/{id}` | pemilik atau admin |
| GET | `/api/users` | admin |
| DELETE | `/api/users/{id}` | admin |
| GET | `/api/audit` | admin |
