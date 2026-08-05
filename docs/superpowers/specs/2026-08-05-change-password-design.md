# Ganti Password Mandiri — Design Specification

Tanggal: 2026-08-05

## Overview

User yang sudah login belum punya cara mengganti passwordnya sendiri. Satu-satunya
jalur penggantian saat ini adalah `PUT /api/admin/users/:id/password`, yang hanya
bisa dipakai admin terhadap user lain dan tidak meminta password lama.

Spesifikasi ini menambahkan endpoint `PUT /api/auth/password` beserta modal di
frontend, dengan password lama sebagai syarat validasi. User hanya bisa mengubah
password miliknya sendiri: id diambil dari klaim JWT, tidak pernah dari body.

## Keputusan

| Keputusan | Pilihan | Alasan |
|---|---|---|
| Penempatan UI | Dropdown pada nama user di navbar → modal | Tanpa route baru; mengikuti pola `Modal` + `ResetPasswordModal` yang sudah ada |
| Sesi setelah sukses | User tetap login, token tidak disentuh | Tidak memutus aktivitas membaca; token lama kedaluwarsa sendiri dalam 24 jam |
| Panjang minimum | 5 karakter | Sama dengan `RegisterRequest`, `AdminUserCreateRequest`, dan `AdminPasswordResetRequest` |
| Password baru == lama | Ditolak backend | Mencegah submit yang tidak berefek apa-apa |
| Konfirmasi password | Hanya di frontend | Backend tidak butuh field ketiga untuk memutuskan apa pun |
| Lapisan logika | Method baru di `AuthService` | Service itu sudah memegang urusan user + password; nol file baru di layer service |
| Status HTTP untuk password lama salah | 400, bukan 401 | Lihat bagian berikut |

### Mengapa bukan 401

Interceptor respons di `frontend/src/api/client.ts` menghapus token dan
mengarahkan browser ke `/login` pada **setiap** respons 401. Kalau "password lama
salah" dijawab 401 seperti `Login`, user yang salah ketik akan terlempar keluar
dari aplikasi alih-alih melihat pesan error di dalam modal.

Karena itu endpoint ini memakai 400 dengan kode error spesifik untuk kegagalan
validasi, dan menyisakan 401 murni untuk kegagalan autentikasi token dari
`AuthMiddleware` — di mana perilaku logout otomatis memang yang diinginkan.

### Alternatif yang ditolak

- **Invalidasi semua token lama** lewat kolom `token_version` di tabel `users`
  yang dicek `AuthMiddleware`. Benar-benar mematikan sesi di semua perangkat,
  tetapi menambah migrasi DB dan satu query per request ke seluruh endpoint
  terproteksi. Tidak sebanding untuk aplikasi baca buku internal.
- **Paksa logout setelah ganti password.** Memutus aktivitas, dan tetap hanya
  membersihkan token di browser yang sedang dipakai — jadi kesan "aman" yang
  diberikannya tidak akurat.
- **`ProfileService` terpisah.** Hanya berisi satu method sekarang, sekaligus
  memecah urusan password ke dua service.
- **Perluas `AdminUserService`.** Salah tempat: service itu untuk aksi admin
  terhadap user lain.

## Kontrak API

`PUT /api/auth/password`, dilindungi `AuthMiddleware`.

```json
// request
{ "old_password": "12345", "new_password": "rahasia-baru" }

// 200 OK
{ "success": true, "data": { "message": "password updated" } }
```

| Kondisi | HTTP | `error.code` | Pesan |
|---|---|---|---|
| Body tidak valid, `new_password` < 5 karakter | 400 | `VALIDATION_ERROR` | detail dari binding |
| Password lama salah | 400 | `INVALID_OLD_PASSWORD` | `old password is incorrect` |
| Password baru sama dengan lama | 400 | `SAME_PASSWORD` | `new password must be different from the old password` |
| Token tidak ada / tidak valid | 401 | dari `AuthMiddleware` | frontend logout otomatis, sesuai harapan |
| Gagal hash atau tulis DB | 500 | `SERVER_ERROR` | pesan error |

## Backend

### `internal/dto/auth_dto.go`

```go
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=5"`
}
```

Tidak ada `binding:"min=5"` pada `OldPassword`: yang menentukan diterima atau
tidaknya adalah kecocokan dengan hash, dan aturan panjang bisa saja berubah
setelah password lama itu dibuat.

### `internal/service/auth_service.go`

Dua sentinel error baru, mengikuti pola `ErrEmailTaken` dkk di
`admin_user_service.go`:

```go
var (
	ErrInvalidOldPassword = errors.New("old password is incorrect")
	ErrSamePassword       = errors.New("new password must be different from the old password")
)
```

Method baru:

```
ChangePassword(ctx, userID, req) error:
  user ← userRepo.GetByID(ctx, userID)          // error diteruskan apa adanya
  if !CheckPassword(user.PasswordHash, req.OldPassword) → ErrInvalidOldPassword
  if  CheckPassword(user.PasswordHash, req.NewPassword) → ErrSamePassword
  hash ← HashPassword(req.NewPassword)
  return userRepo.UpdatePassword(ctx, userID, hash)
```

Pemeriksaan "password baru sama dengan lama" memakai `CheckPassword` terhadap
hash lama, bukan perbandingan string terhadap `req.OldPassword`. Keduanya
ekuivalen di sini, tetapi bentuk ini tetap benar seandainya nanti password lama
tidak lagi wajib dikirim.

### Interface repository demi keteruji-an

Field `userRepo` pada `AuthService` bertipe konkret `*repository.UserRepository`,
sehingga service tidak bisa diuji tanpa Postgres sungguhan. Tipe field dan
parameter konstruktor diganti menjadi interface yang dideklarasikan di package
`service`:

```go
type UserStore interface {
	Create(ctx context.Context, user *model.User) error
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id string) (*model.User, error)
	UpdatePassword(ctx context.Context, id, hash string) error
}
```

Empat method itu persis yang dipakai `AuthService` setelah perubahan ini.
`*repository.UserRepository` memenuhinya tanpa deklarasi tambahan, jadi
pemanggil `NewAuthService` di `cmd/server/main.go` tidak berubah.

`AdminUserService` tetap memakai `*repository.UserRepository` konkret — ia
memakai method yang lebih banyak dan tidak termasuk lingkup pekerjaan ini.

### `internal/handler/auth_handler.go`

Handler `ChangePassword` membaca `c.GetString("user_id")`, bind body, panggil
service, lalu memetakan sentinel error ke tabel kontrak API di atas dengan
`errors.Is`. Sukses dibalas `utils.SuccessResponse(c, http.StatusOK, gin.H{"message": "password updated"})`.

### `internal/router/router.go`

```go
auth.PUT("/password", middleware.AuthMiddleware(cfg.JWTSecret), cfg.AuthHandler.ChangePassword)
```

Ditempatkan di grup `auth`, memakai bentuk middleware per-route yang sama dengan
`auth.GET("/me", ...)`.

## Frontend

### `src/api/auth.ts`

```ts
changePassword: async (oldPassword: string, newPassword: string) => {
  await client.put('/auth/password', { old_password: oldPassword, new_password: newPassword });
}
```

### `src/components/account/ChangePasswordModal.tsx` (baru)

Direktori `account/` baru, sejajar dengan `admin/`, `common/`, `ebook/`,
`layout/`, dan `reader/`: modal ini adalah aksi user terhadap akunnya sendiri,
bukan aksi admin dan bukan komponen generik.

Membungkus `components/common/Modal`, dengan bentuk yang mengikuti
`admin/ResetPasswordModal.tsx`: state `loading` dan `error`, field direset lewat
`useEffect` saat modal dibuka, pesan error diambil dari
`err.response.data.error.message` dengan teks fallback.

Teks antarmuka memakai bahasa Inggris, mengikuti seluruh UI yang sudah ada.

Isi form: **Current Password**, **New Password** (`minLength={5}`, disertai
keterangan "Minimum 5 characters."), dan **Confirm New Password**. Sebelum
memanggil API, form memeriksa `newPassword === confirmPassword` dan menampilkan
"Password confirmation does not match" tanpa request kalau berbeda.

Setelah sukses, form diganti pesan hijau "Password changed successfully." dan
modal menutup sendiri setelah 1,5 detik; menutupnya lebih awal lewat tombol
Cancel atau backdrop juga boleh. Timer dibersihkan saat komponen unmount dan
saat modal dibuka kembali, supaya timer dari submit sebelumnya tidak menutup
modal yang baru dibuka. Tidak ada token, `localStorage`, maupun state
`AuthContext` yang disentuh — user tetap login.

### `src/components/layout/Navbar.tsx`

Pada layout desktop, nama user menjadi tombol pembuka dropdown berisi "Change
Password" dan "Logout"; tombol logout yang berdiri sendiri dipindahkan ke
dalamnya. Dropdown ditutup lewat overlay `fixed inset-0` transparan, idiom yang
sama dengan backdrop di `Modal.tsx`, dan juga ditutup saat salah satu itemnya
dipilih.

Pada menu mobile, "Change Password" ditambahkan ke blok bawah yang sudah memuat
nama user dan tombol logout. Membukanya menutup menu mobile.

Navbar memegang state `isChangePasswordOpen` dan merender `ChangePasswordModal`.

## Testing

**Backend** — `internal/service/auth_service_test.go` baru, memakai fake
`UserStore` dan `testify` yang sudah ada di `go.mod`. Kasus:

1. Sukses — store menerima hash baru, hash itu lolos `CheckPassword` dengan
   password baru dan gagal dengan password lama.
2. Password lama salah — mengembalikan `ErrInvalidOldPassword`, `UpdatePassword`
   tidak pernah dipanggil.
3. Password baru sama dengan lama — mengembalikan `ErrSamePassword`,
   `UpdatePassword` tidak pernah dipanggil.
4. User tidak ditemukan — error dari store diteruskan.

**Frontend** — repo belum punya test runner sama sekali; menambah Vitest hanya
untuk satu modal ada di luar lingkup. Verifikasi manual:

1. Ganti password → logout → login dengan password baru berhasil.
2. Password lama salah menampilkan error di dalam modal dan **tidak** melempar
   user ke `/login`.
3. Konfirmasi tidak cocok tertahan di frontend, tanpa request ke backend.
4. Password baru sama dengan lama ditolak dengan pesan dari backend.
5. Dropdown navbar dan menu mobile membuka modal serta menutup dengan benar.

## Di luar lingkup

- Mengubah nama atau email sendiri.
- Reset password lewat email untuk user yang lupa password.
- Pembatasan laju percobaan pada endpoint ini — pemanggilnya sudah terautentikasi
  dan hanya bisa menebak password miliknya sendiri.
- Menaikkan panjang minimum password di seluruh aplikasi.
