//! Layanan keamanan NexusStack: hashing Argon2id + audit log.
use argon2::{
    password_hash::{rand_core::OsRng, PasswordHash, PasswordHasher, PasswordVerifier, SaltString},
    Argon2,
};
use axum::{
    extract::State,
    routing::{get, post},
    Json, Router,
};
use serde::{Deserialize, Serialize};
use std::{
    sync::{Arc, Mutex},
    time::{SystemTime, UNIX_EPOCH},
};

#[derive(Clone, Default)]
struct AppState {
    audit: Arc<Mutex<Vec<AuditEntry>>>,
}

#[derive(Serialize, Deserialize, Clone)]
struct AuditEntry {
    #[serde(default)]
    ts: u64,
    actor: String,
    action: String,
    #[serde(default)]
    detail: String,
}

#[derive(Deserialize)]
struct HashReq {
    password: String,
}
#[derive(Serialize)]
struct HashRes {
    hash: String,
}
#[derive(Deserialize)]
struct VerifyReq {
    password: String,
    hash: String,
}
#[derive(Serialize)]
struct VerifyRes {
    ok: bool,
}

async fn hash(Json(req): Json<HashReq>) -> Json<HashRes> {
    // Argon2 berat di CPU, jalankan di thread blocking.
    let h = tokio::task::spawn_blocking(move || {
        let salt = SaltString::generate(&mut OsRng);
        Argon2::default()
            .hash_password(req.password.as_bytes(), &salt)
            .map(|h| h.to_string())
            .unwrap_or_default()
    })
    .await
    .unwrap_or_default();
    Json(HashRes { hash: h })
}

async fn verify(Json(req): Json<VerifyReq>) -> Json<VerifyRes> {
    let ok = tokio::task::spawn_blocking(move || {
        PasswordHash::new(&req.hash)
            .map(|p| Argon2::default().verify_password(req.password.as_bytes(), &p).is_ok())
            .unwrap_or(false)
    })
    .await
    .unwrap_or(false);
    Json(VerifyRes { ok })
}

async fn add_audit(State(s): State<AppState>, Json(mut e): Json<AuditEntry>) -> &'static str {
    e.ts = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or(0);
    let mut log = s.audit.lock().unwrap();
    log.push(e);
    if log.len() > 500 {
        log.remove(0); // batasi memori
    }
    "ok"
}

async fn list_audit(State(s): State<AppState>) -> Json<Vec<AuditEntry>> {
    let log = s.audit.lock().unwrap();
    Json(log.iter().rev().take(50).cloned().collect())
}

#[tokio::main]
async fn main() {
    let app = Router::new()
        .route("/hash", post(hash))
        .route("/verify", post(verify))
        .route("/audit", post(add_audit).get(list_audit))
        .route("/healthz", get(|| async { "ok" }))
        .with_state(AppState::default());

    let addr = std::env::var("ADDR").unwrap_or_else(|_| "0.0.0.0:8081".into());
    println!("NexusStack hasher (Rust) berjalan di {addr}");
    let listener = tokio::net::TcpListener::bind(addr).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}
