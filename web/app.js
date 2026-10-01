const $ = (id) => document.getElementById(id);
let mode = "login";
let me = null;

const api = async (path, opts = {}) => {
  const token = localStorage.getItem("token");
  const res = await fetch("/api" + path, {
    ...opts,
    headers: { "Content-Type": "application/json", ...(token && { Authorization: "Bearer " + token }) },
  });
  if (res.status === 204) return null;
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    if (res.status === 401 && token) logout();
    throw new Error(data.error || "Terjadi kesalahan.");
  }
  return data;
};

function toast(t) {
  const el = $("toast");
  el.textContent = t;
  el.classList.add("show");
  setTimeout(() => el.classList.remove("show"), 2200);
}

function setMode(m) {
  mode = m;
  $("tab-login").classList.toggle("on", m === "login");
  $("tab-register").classList.toggle("on", m === "register");
  $("auth-submit").textContent = m === "login" ? "Masuk" : "Buat akun";
  $("password").autocomplete = m === "login" ? "current-password" : "new-password";
  $("auth-msg").textContent = "";
}
$("tab-login").onclick = () => setMode("login");
$("tab-register").onclick = () => setMode("register");

$("auth-form").onsubmit = async (e) => {
  e.preventDefault();
  try {
    const d = await api("/auth/" + mode, {
      method: "POST",
      body: JSON.stringify({ username: $("username").value, password: $("password").value }),
    });
    localStorage.setItem("token", d.token);
    $("password").value = "";
    start();
  } catch (err) {
    $("auth-msg").textContent = err.message;
  }
};

function logout() {
  localStorage.removeItem("token");
  me = null;
  $("app").hidden = true;
  $("auth").hidden = false;
}
$("logout").onclick = logout;

// Membuat elemen tanpa innerHTML agar aman dari XSS.
function el(tag, props = {}, ...kids) {
  const n = Object.assign(document.createElement(tag), props);
  kids.forEach((k) => n.append(k));
  return n;
}
function empty(ul, text) { ul.replaceChildren(el("li", { className: "empty" }, text)); }

async function start() {
  try {
    me = await api("/me");
  } catch { return logout(); }
  $("auth").hidden = true;
  $("app").hidden = false;
  $("who").textContent = `${me.usr} (${me.role})`;
  $("admin").hidden = me.role !== "admin";
  loadNotes();
  if (me.role === "admin") { loadUsers(); loadAudit(); }
}

async function loadNotes() {
  const ul = $("notes");
  const notes = await api("/notes");
  if (!notes.length) return empty(ul, "Belum ada catatan. Tulis yang pertama di atas.");
  ul.replaceChildren(...notes.map((n) => {
    const del = el("button", { className: "danger", textContent: "Hapus" });
    del.onclick = async () => {
      if (!confirm(`Hapus catatan "${n.title}"?`)) return;
      try { await api("/notes/" + n.id, { method: "DELETE" }); toast("Catatan dihapus"); loadNotes(); loadAudit(); }
      catch (e) { toast(e.message); }
    };
    return el("li", {}, el("div", {}, el("strong", { textContent: n.title }), el("p", { textContent: n.body || "—" })), del);
  }));
}

$("note-form").onsubmit = async (e) => {
  e.preventDefault();
  try {
    await api("/notes", { method: "POST", body: JSON.stringify({ title: $("note-title").value, body: $("note-body").value }) });
    e.target.reset();
    toast("Catatan disimpan");
    loadNotes(); loadAudit();
  } catch (err) { toast(err.message); }
};

async function loadUsers() {
  const users = await api("/users");
  $("users").replaceChildren(...users.map((u) => {
    const label = el("strong", { textContent: u.username });
    if (u.role === "admin") label.append(el("span", { className: "badge", textContent: "admin" }));
    const row = el("li", {}, label);
    if (u.id !== me.sub) {
      const del = el("button", { className: "danger", textContent: "Hapus pengguna" });
      del.onclick = async () => {
        if (!confirm(`Hapus pengguna "${u.username}" beserta semua catatannya?`)) return;
        try { await api("/users/" + u.id, { method: "DELETE" }); toast("Pengguna dihapus"); loadUsers(); loadAudit(); }
        catch (e) { toast(e.message); }
      };
      row.append(del);
    }
    return row;
  }));
}

async function loadAudit() {
  if (me?.role !== "admin") return;
  try {
    const log = await api("/audit");
    if (!log.length) return empty($("audit"), "Belum ada aktivitas tercatat.");
    $("audit").replaceChildren(...log.map((a) =>
      el("li", {}, `${new Date(a.ts * 1000).toLocaleTimeString("id-ID")}  ${a.actor}  ${a.action}  ${a.detail}`)));
  } catch { empty($("audit"), "Layanan Rust belum berjalan."); }
}

if (localStorage.getItem("token")) start();
