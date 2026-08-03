# Migrasi PostgreSQL lokal → Supabase — Design Specification

Tanggal: 2026-08-03

## Overview

Memindahkan database dari container PostgreSQL lokal ke PostgreSQL terkelola di
Supabase, lalu mencabut service `postgres` dari Docker Compose. Redis tetap
berjalan lokal.

## Keputusan

| Keputusan | Pilihan | Alasan |
|---|---|---|
| Data lama | Tidak dimigrasikan, mulai bersih | Skema dibuat oleh migrasi saat startup; user admin di-seed ulang |
| Mode koneksi | Session pooler, port 5432 | Tersedia lewat IPv4 sehingga container Docker bisa menjangkaunya; mode session mendukung prepared statement yang dipakai `lib/pq` secara implisit untuk setiap query berparameter |
| Bentuk config | Satu `DATABASE_URL` | Salin-tempel langsung dari dashboard; menghindari escaping manual pada DSN key=value |
| Redis | Tetap lokal | Di luar cakupan permintaan; isinya hanya cache progres baca yang boleh hilang |

### Mode koneksi yang ditolak

- **Koneksi langsung** (`db.<ref>.supabase.co:5432`) — kini hanya IPv6 kecuali
  ada add-on IPv4. Docker tanpa IPv6 gagal dengan gejala resolusi yang
  membingungkan.
- **Transaction pooler** (port 6543) — mode transaksi tidak mendukung prepared
  statement, sehingga query berparameter `lib/pq` gagal kecuali driver dipaksa
  ke protokol sederhana atau diganti ke `pgx`. Tidak ada manfaatnya untuk
  backend yang berjalan terus-menerus.

## Konfigurasi

Sembilan variabel dihapus — enam `DB_*` yang dibaca backend dan tiga
`POSTGRES_*` yang hanya dipakai container postgres — diganti satu:

```dotenv
DATABASE_URL=postgresql://postgres.<project-ref>:<password>@aws-0-<region>.pooler.supabase.com:5432/postgres
```

Di `Config`, enam field `DB*` menjadi satu `DatabaseURL` yang diambil lewat
`mustEnv`. Method `DBDSN()` dihapus: satu-satunya tugasnya merangkai enam bagian
menjadi DSN key=value, sedangkan `lib/pq` menerima bentuk URL apa adanya di
`sql.Open`.

`databaseURL()` memvalidasi skema (`postgres://` atau `postgresql://`), sejenis
`r2Endpoint()`. Dashboard Supabase menampilkan beberapa varian string koneksi di
tab berbeda, dan menyalin yang keliru menghasilkan kegagalan driver yang tidak
informatif.

SSL tidak memerlukan variabel terpisah: `lib/pq` memakai `sslmode=require`
sebagai default — berbeda dari libpq C yang memakai `prefer` — sehingga URL
Supabase yang tidak mencantumkan `sslmode` tetap tersambung terenkripsi.

## Lapisan database

`pkg/database/postgres.go` berubah di dua titik:

1. `sql.Open("postgres", cfg.DatabaseURL)`.
2. Batas umur koneksi ditambahkan:

```go
db.SetConnMaxLifetime(30 * time.Minute)
db.SetConnMaxIdleTime(5 * time.Minute)
```

Sebelumnya database berada satu jaringan Docker, jadi koneksi menganggur
praktis abadi. Di balik pooler Supabase, koneksi menganggur diputus secara
sepihak; tanpa batas umur, `database/sql` menyerahkan koneksi mati ke query
berikutnya dan gejalanya muncul sebagai `unexpected EOF` yang sporadis.

`SetMaxOpenConns(25)` dipertahankan — masih di bawah kuota pooler.

## Migrasi

Tetap dijalankan otomatis saat startup. Ketiga berkas idempoten (`IF NOT EXISTS`
pada setiap pernyataan), jadi aman diulang.

`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"` membutuhkan hak elevated. Peran
`postgres.<project-ref>` pada session pooler memilikinya, dan Supabase umumnya
sudah memasang ekstensi tersebut di schema `extensions`, sehingga pernyataannya
menjadi no-op.

## Docker Compose

Service `postgres`, volume `pgdata`, dan `depends_on: postgres` dicabut. Backend
menukar enam `DB_*` dengan satu `DATABASE_URL`. Menyisakan empat service: redis,
backend, frontend, nginx.

## Langkah manual di Supabase (sekali)

1. Buat project Supabase.
2. **Connect** → **Session pooler** → salin URI.
3. Ganti `[YOUR-PASSWORD]` dengan password database, lalu isikan ke
   `DATABASE_URL` di `.env`.

## Verifikasi

Statis: `go build ./...`, `go vet ./...`, `go test ./...`, dan
`docker compose config` tanpa warning.

End-to-end terhadap project Supabase sungguhan:

| Langkah | Harapan |
|---|---|
| Startup backend | `Connected to PostgreSQL`, ketiga migrasi ter-apply, `Default admin user ready` |
| Tabel di Supabase | `users`, `ebooks`, `reading_progress`, `bookmarks`, `history` ada |
| Login | 200 |
| Upload PDF | 200, baris baru di `ebooks` |
| Baca ebook | progres tersimpan |
| Hapus ebook | 200, baris hilang |

## Jebakan yang ditemukan saat implementasi

Docker Compose melakukan interpolasi variabel pada nilai di `.env`. Password
yang mengandung `$` membuat bagian sesudahnya diperlakukan sebagai nama
variabel dan dikosongkan, sehingga URL yang sampai ke backend rusak — gejalanya
`lib/pq` membaca project-ref sebagai nama host dan gagal dengan
`no such host`. Lebih buruk lagi, potongan password ikut tercetak di peringatan
Compose sehingga masuk ke log.

Penyelesaian saat itu: password diganti dengan yang tidak mengandung karakter
khusus. Alternatif yang setara: percent-encode `$` menjadi `%24`, atau
menuliskannya `$$` di `.env`. Catatan ini ditambahkan ke `.env.example`.

## Risiko yang diketahui, di luar cakupan

Migrasi dijalankan setiap backend start tanpa tabel pelacak versi. Selama
database bersifat lokal dan sekali pakai, ini tidak berbahaya. Setelah pindah ke
Supabase, database itu menjadi satu-satunya dan permanen — migrasi
non-idempoten yang ditambahkan kemudian akan langsung mengenai data sungguhan.
Menambahkan tabel `schema_migrations` adalah perbaikan yang tepat, dicatat
sebagai tindak lanjut terpisah.
