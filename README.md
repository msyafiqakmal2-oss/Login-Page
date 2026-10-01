# NexusStack

Proyek berlapis dengan beberapa bahasa pemrograman:

```
nexusstack/
├── gateway/   Go    – API, JWT, CRUD, menyajikan frontend
├── hasher/    Rust  – hashing Argon2id dan audit log
├── web/       JS    – halaman login, dashboard, hapus catatan/pengguna
├── docs/      panduan upgrade
├── docker-compose.yml, Makefile
```

## Menjalankan

**Cara cepat (hanya Go, tanpa Rust):**
```bash
make dev          # buka http://localhost:8080
```

**Lengkap (Go + Rust), dua terminal:**
```bash
make rust         # layanan Rust di :8081
make go           # gateway Go di :8080
```

**Docker:**
```bash
cp .env.example .env && make up
```

Daftar akun pertama, otomatis menjadi **admin** dan bisa menghapus pengguna serta melihat log audit.

## Catatan keamanan
- Ganti `JWT_SECRET` sebelum produksi.
- Mode tanpa Rust memakai hash sederhana, hanya untuk pengembangan.
- Data masih in-memory (hilang saat restart); lihat `docs/UPGRADE.md`.
