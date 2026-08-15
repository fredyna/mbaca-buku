# Log Aktivitas User — Desain

Tanggal: 2026-08-11

## Tujuan

Admin bisa melihat, untuk setiap user: kapan terakhir login, kapan terakhir
aktif, serta sistem operasi dan browser yang dipakai.

Riwayat lengkap setiap event disimpan di database (bukan hanya nilai terakhir),
supaya halaman detail riwayat per user bisa ditambahkan nanti tanpa migrasi
data. UI pada iterasi ini terbatas pada kolom di daftar admin users.

## Batasan yang memengaruhi desain

`database.RunMigrations` menjalankan ulang **semua** file `migrations/*.sql`
pada setiap startup — tidak ada tabel versi migrasi. Setiap pernyataan dalam
migrasi baru wajib idempotent.

Migrasi 005 mencatat bahwa lib/pq tidak bisa memindai `NULL` ke `string` Go.
Semua kolom teks pada tabel baru memakai `NOT NULL DEFAULT ''`.

Aplikasi berjalan di belakang nginx, yang sudah mengirim `X-Real-IP` dan
`X-Forwarded-For`, sehingga `c.ClientIP()` menghasilkan IP asli klien.

Redis sudah tersedia (`pkg/cache`, dipakai `ReadingService`) dan dipakai di sini
sebagai throttle.

## Model data

Migrasi `backend/migrations/006_add_user_activity_logs.sql`:

```sql
CREATE TABLE IF NOT EXISTS user_activity_logs (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event      VARCHAR(20) NOT NULL,              -- 'login' | 'active'
    os         VARCHAR(50)  NOT NULL DEFAULT '',
    browser    VARCHAR(50)  NOT NULL DEFAULT '',
    device     VARCHAR(20)  NOT NULL DEFAULT '',  -- desktop | mobile | tablet
    ip_address VARCHAR(45)  NOT NULL DEFAULT '',  -- 45 = panjang maksimum IPv6
    user_agent TEXT         NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_activity_user_created ON user_activity_logs(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_activity_user_event   ON user_activity_logs(user_id, event, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_activity_created      ON user_activity_logs(created_at);
```

`ON DELETE CASCADE`: menghapus user menghapus lognya, mengikuti pola tabel
`history`, `bookmarks`, dan `reading_progress`.

Dua index pertama melayani query daftar admin (baris terakhir per user, dan
baris login terakhir per user); index ketiga melayani penghapusan berbasis
retensi.

`user_agent` mentah disimpan sebagai cadangan: kalau parser salah mengenali
sebuah perangkat, datanya masih bisa diperiksa dan diperbaiki tanpa kehilangan
riwayat.

`ip_address` disimpan untuk keperluan audit keamanan.

## Perekaman event

### Event `login`

Dicatat di **handler**, bukan di `AuthService`:

```
POST /api/auth/login    → AuthHandler.Login
POST /api/auth/oauth    → AuthHandler.OAuth
POST /api/auth/register → AuthHandler.Register
                            authService.X(...)  → resp berhasil
                            activityService.Record(ctx, resp.User.ID, "login", MetaFrom(c))
```

`AuthService` saat ini bebas dari HTTP dan diuji dengan fake `UserStore`.
Handler sudah memegang `*gin.Context` (sumber User-Agent dan IP) serta
`resp.User.ID`, jadi mencatat di lapisan handler tidak menambah dependency
maupun perubahan test pada service auth.

Registrasi dihitung sebagai login karena `Register` langsung mengembalikan token
— user tersebut memang mulai bersesi saat itu.

### Event `active`

Dicatat di `AuthMiddleware` setelah token terbukti valid:

```
AuthMiddleware → ParseToken sukses, user_id diketahui
   SET activity:seen:<user_id> 1 NX EX 300
     ├─ berhasil (key belum ada) → catat 1 baris 'active'
     └─ gagal   (key masih ada)  → lewati, tidak menyentuh Postgres
   c.Next()
```

Throttle 5 menit per user. Saat user idle tidak ada baris yang ditulis sama
sekali; saat aktif maksimum ~12 baris/jam/user. Throttle memakai Redis sehingga
jalur request yang di-skip tidak melakukan query database apa pun.

Penulisan baris dilakukan di goroutine terpisah agar tidak menambah latensi
request. `RequestMeta` diambil dari context gin **sebelum** goroutine dijalankan,
karena `*gin.Context` tidak boleh diakses setelah request selesai; goroutine
memakai `context.Background()` dengan timeout sendiri. Jumlah goroutine terbatas
dengan sendirinya oleh throttle.

Kegagalan Redis atau kegagalan insert dicatat ke log server dan diabaikan.
Pencatatan aktivitas tidak boleh pernah menggagalkan request user.

`AuthMiddleware(jwtSecret)` menjadi `AuthMiddleware(jwtSecret, recorder)`.
`recorder` bernilai `nil` diperbolehkan dan mematikan pencatatan, sehingga test
middleware yang ada tetap bisa membangun middleware tanpa dependency.

## Unit baru

| Unit | Tanggung jawab | Antarmuka | Dependency |
|---|---|---|---|
| `pkg/useragent/parse.go` | mengurai string User-Agent | `Parse(ua string) Info` | tidak ada |
| `internal/model/activity.go` | struct baris log dan metadata request | `UserActivity`, `RequestMeta` | tidak ada |
| `internal/repository/activity_repo.go` | akses tabel | `Insert`, `DeleteOlderThan` | `*sql.DB` |
| `internal/service/activity_service.go` | throttle, perekaman, cleanup | `Record`, `StartCleanup` | repo, `Throttler` |

### `pkg/useragent`

```go
type Info struct {
    Browser string // "Chrome 128"
    OS      string // "macOS"
    Device  string // "desktop" | "mobile" | "tablet"
}

func Parse(ua string) Info
```

Fungsi murni tanpa dependency eksternal. Mengenali browser umum (Chrome, Safari,
Firefox, Edge, Opera, Samsung Internet) dan OS umum (Windows, macOS, Linux,
Android, iOS). String yang tidak dikenali menghasilkan field kosong, bukan
error — log tetap tertulis dengan `user_agent` mentah yang bisa diperiksa.

Urutan pemeriksaan penting: Edge dan Opera memuat "Chrome" pada UA-nya, dan
Chrome memuat "Safari", sehingga yang lebih spesifik harus diperiksa lebih
dulu.

### `ActivityService`

```go
type Throttler interface {
    Allow(ctx context.Context, key string, window time.Duration) bool
}
```

Service bergantung pada interface ini, bukan pada `*redis.Client`, sehingga
perilaku throttle bisa diuji tanpa Redis yang hidup. Implementasi produksi
membungkus `SetNX`.

Metadata request diekstrak di lapisan HTTP, bukan di service — service tetap
bebas dari gin, seperti semua service lain di project ini:

```go
// internal/model/activity.go
type RequestMeta struct {
    IP        string
    UserAgent string
}

// dipanggil dari handler/middleware
func MetaFrom(c *gin.Context) model.RequestMeta

// internal/service/activity_service.go
func (s *ActivityService) Record(ctx context.Context, userID, event string, meta model.RequestMeta)
```

`Record` mengurai `meta.UserAgent`, lalu untuk event `active` memeriksa throttle
sebelum menulis. Event `login` selalu ditulis — setiap login adalah peristiwa
yang berbeda dan jumlahnya sudah terbatas dengan sendirinya.

Karena `Record` menerima `context.Context` dan `RequestMeta` (bukan
`*gin.Context`), goroutine pencatat bisa memanggilnya dengan
`context.Background()` secara aman, dan testnya tidak perlu membangun request
HTTP sama sekali.

`StartCleanup(ctx)` mengikuti pola `readingService.StartFlusher`: menghapus
sekali saat startup, lalu setiap 24 jam, sampai context dibatalkan.

## Query daftar admin

Satu query, tanpa N+1:

```sql
SELECT u.id, u.name, u.email, u.role, u.created_at, u.updated_at,
       last.created_at, last.os, last.browser,
       login.created_at
FROM users u
LEFT JOIN LATERAL (
    SELECT created_at, os, browser FROM user_activity_logs
    WHERE user_id = u.id ORDER BY created_at DESC LIMIT 1
) last ON TRUE
LEFT JOIN LATERAL (
    SELECT created_at FROM user_activity_logs
    WHERE user_id = u.id AND event = 'login' ORDER BY created_at DESC LIMIT 1
) login ON TRUE
ORDER BY u.created_at DESC
LIMIT $1 OFFSET $2
```

`LEFT JOIN LATERAL` dipakai karena yang dibutuhkan adalah *baris* terakhir, bukan
nilai agregat; bentuk ini memakai index `(user_id, created_at DESC)` secara
langsung. `LEFT` menjaga user yang belum pernah punya log tetap muncul.

OS dan browser diambil dari aktivitas terakhir apa pun, bukan dari login
terakhir — pertanyaan yang dijawab kolom itu adalah "perangkat apa yang dipakai
sekarang".

Kolom waktu dipindai ke `sql.NullTime` dan diekspos sebagai `*time.Time`,
sehingga user yang belum pernah login tampil `null` dan bukan
`0001-01-01T00:00:00Z`.

`UserRepository.List` adalah satu-satunya pemanggil query ini, jadi perubahan
terbatas pada metode tersebut.

## API

Tidak ada route baru. `GET /api/admin/users` tetap sama; `AdminUserResponse`
bertambah empat field:

```go
LastLoginAt  *time.Time `json:"last_login_at"`
LastActiveAt *time.Time `json:"last_active_at"`
LastOS       string     `json:"last_os"`
LastBrowser  string     `json:"last_browser"`
```

## UI

`frontend/src/pages/UsersPage.tsx` mendapat kolom "Login terakhir" dan
"Aktivitas terakhir". "Joined" turun menjadi baris kecil di bawah email agar
tabel tidak menjadi tujuh kolom sempit:

```
Nama            Role    Login terakhir   Aktivitas terakhir   Aksi
Fredy (you)     admin   11 Ags, 09:14    ● baru saja          Edit ...
f@mail.com                               Chrome 128 · macOS
Joined 24 Jul

Budi            user    10 Ags, 20:01    3 hari lalu          Edit ...
b@mail.com                               Chrome 128 · Android
Joined 02 Ags

Siti            user    —                belum pernah login   Edit ...
s@mail.com
Joined 09 Ags
```

Titik hijau muncul bila aktivitas terakhir kurang dari 5 menit — sama dengan
window throttle, sehingga user yang sedang aktif tidak pernah tampil sebagai
tidak aktif.

Kartu mobile mendapat dua baris informasi yang sama.

`frontend/src/utils/time.ts` baru berisi `formatRelativeTime` dan
`formatDateTime`; `AdminUser` di `api/adminUsers.ts` mendapat empat field baru
sebagai `string | null`.

## Retensi

Log lebih tua dari 90 hari dihapus:

```sql
DELETE FROM user_activity_logs WHERE created_at < NOW() - INTERVAL '90 days'
```

Dijalankan oleh `StartCleanup` di dalam proses backend — sekali saat startup,
lalu setiap 24 jam. Tidak perlu cron eksternal, dan restart container tidak
membuat cleanup terlewat.

Perkiraan ukuran pada 20 user aktif: ~36 baris/user/hari, sekitar 65.000 baris
pada steady state — beberapa MB.

## Testing

- `pkg/useragent/parse_test.go` — table-driven dengan string UA asli:
  Chrome/macOS, Chrome/Android, Safari/iOS, Firefox/Windows, Edge/Windows,
  Samsung Internet, UA kosong, UA sampah. Unit ini paling rawan salah dan paling
  murah diuji.
- `internal/service/activity_service_test.go` — dengan fake `Throttler` dan fake
  repo: request kedua dalam window tidak menulis baris; event `login` selalu
  menulis; error dari throttler maupun repo tidak dipropagasi menjadi error
  request.
- `internal/middleware/auth_test.go` — recorder `nil` tidak menyebabkan panik;
  token valid memanggil recorder tepat sekali; token tidak valid tidak memanggil
  recorder.

Test repository dilewati: test yang ada di `internal/repository/` hanya menguji
fungsi murni (`normalizeEbookFilter`) dan project ini tidak memiliki
infrastruktur database untuk test. Query SQL diverifikasi dengan menjalankan
aplikasi.

## Di luar lingkup

Sengaja tidak dibuat pada iterasi ini:

- Endpoint dan UI riwayat detail per user
- Halaman aktivitas login untuk user biasa
- Deteksi bot/crawler
- Geolokasi dari IP
- Kolom snapshot di tabel `users`

Data sudah tersimpan lengkap, sehingga masing-masing bisa ditambahkan nanti
tanpa perubahan skema.
