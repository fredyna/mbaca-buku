# Migrasi MinIO → Cloudflare R2 — Design Specification

Tanggal: 2026-08-03

## Overview

Mengganti MinIO sebagai object storage PDF dengan Cloudflare R2. MinIO dicabut
sepenuhnya — tidak ada lagi container storage yang dijalankan sendiri, baik di
production maupun di development lokal.

## Keputusan

| Keputusan | Pilihan | Alasan |
|---|---|---|
| Cakupan | Hapus MinIO total, R2 di semua environment | Satu jalur kode; tidak ada perbedaan perilaku dev vs prod yang harus dijaga konsisten |
| Data lama | Tidak dimigrasikan | Bucket MinIO hanya berisi data uji coba |
| Pengiriman PDF | Presigned URL ditulis ulang ke path same-origin, di-proxy nginx | Perubahan terkecil dari pola yang sudah ada; tidak perlu aturan CORS di bucket; account ID tidak terlihat browser |
| SDK | Tetap `minio-go/v7`, endpoint diarahkan ke R2 | R2 mendukung API S3. `minio.Object` sudah menyediakan `io.ReadSeekCloser` yang dibutuhkan `pdfutil.PageCount`; dengan AWS SDK v2 itu harus ditulis manual di atas ranged GET |
| Port nginx | 80 → 6900 (host) | Permintaan terpisah; container tetap `listen 80` |

## Konfigurasi

Lima variabel `MINIO_*` diganti empat variabel R2:

| Env var | Contoh | Catatan |
|---|---|---|
| `R2_ENDPOINT` | `<account-id>.r2.cloudflarestorage.com` | Host saja — tanpa `https://` dan tanpa akhiran `/<bucket>` |
| `R2_ACCESS_KEY_ID` | 32 karakter heksadesimal | |
| `R2_SECRET_ACCESS_KEY` | 64 karakter heksadesimal | |
| `R2_BUCKET` | `development` | |

`MINIO_USE_SSL` dihapus: R2 selalu TLS.

Keempatnya wajib. Default seperti `localhost:9000` dulu masuk akal untuk MinIO,
tapi untuk R2 nilai default apa pun hanya menunda kegagalan sampai upload
pertama — jadi `mustEnv` menggagalkan startup, mengikuti pola `JWT_SECRET`
yang sudah ada.

`R2_ENDPOINT` divalidasi terpisah oleh `r2Endpoint()`. Dashboard Cloudflare
menampilkan S3 API sebagai URL lengkap dengan nama bucket di belakang, dan
bentuk itu merusak dua tempat sekaligus: SDK menolaknya dengan pesan yang tidak
informatif, dan `proxy_pass` nginx jadi tidak valid. Validasi menolaknya lebih
awal dengan pesan yang menyebut persis apa yang harus dibuang.

## Lapisan storage

`internal/storage/minio.go` → `internal/storage/r2.go`; `MinIOStorage` →
`R2Storage`; `NewMinIOClient` → `NewR2Client`. Keempat method
(`UploadFile`, `GetPresignedURL`, `OpenFile`, `DeleteFile`) mempertahankan
signature, sehingga `EbookService` hanya berganti nama tipe.

Tiga perubahan perilaku:

- `minio.New` dipanggil dengan `Secure: true` dan `Region: "auto"`. R2 hanya
  menerima region `"auto"`, dan menyetelnya eksplisit membuat minio-go melewati
  panggilan GetBucketLocation yang tidak diimplementasikan R2.
- `MakeBucket` dibuang. Bucket dibuat sekali lewat dashboard; token R2 yang
  di-scope ke satu bucket tidak berwenang membuat bucket, jadi auto-create hanya
  akan gagal. `BucketExists` dipertahankan sebagai pemeriksaan fail-fast.
- `GetPresignedURL` menulis ulang prefix `https://<endpoint>` menjadi `/r2`
  (konstanta `storage.URLPrefix`).

## nginx

`nginx/nginx.conf` menjadi `nginx/default.conf.template`, di-mount ke
`/etc/nginx/templates/` supaya image `nginx:alpine` menjalankan `envsubst` saat
start. Ini diperlukan karena account ID berbeda per deployment.

```nginx
location /r2/ {
    proxy_pass https://${R2_ENDPOINT}/;
    proxy_set_header Host ${R2_ENDPOINT};
    proxy_ssl_server_name on;
}
```

Dua hal yang wajib benar: header `Host` harus persis host R2 karena tanda tangan
SigV4 mencakupnya, dan `proxy_ssl_server_name on` supaya SNI terkirim ke
Cloudflare.

`NGINX_ENVSUBST_FILTER=R2_` membatasi substitusi ke variabel `R2_*` saja,
sehingga `$host`, `$uri`, dan `$remote_addr` milik nginx tidak ikut dikosongkan.

**Trade-off yang diambil sadar:** `proxy_pass` dibiarkan literal (hasil
envsubst) alih-alih memakai variabel + `resolver`. Bentuk variabel membuat nginx
bekerja pada URI yang sudah di-decode, yang berpotensi merusak tanda tangan
SigV4 untuk key berkarakter khusus. Konsekuensinya nginx me-resolve DNS R2 sekali
saat startup; jika IP Cloudflare berubah, restart container nginx
menyelesaikannya. Risiko ini kecil karena endpoint R2 bersifat anycast.

## Docker Compose

Service `minio` dan volume `minio_data` dihapus, termasuk `depends_on` di
backend. Service nginx menerima `R2_ENDPOINT` dan `NGINX_ENVSUBST_FILTER`, serta
mount template menggantikan mount `nginx.conf`.

## Langkah manual di Cloudflare (sekali)

1. Buat bucket R2.
2. **Manage API Tokens** → token *Object Read & Write* yang di-scope ke bucket itu.
3. Salin Access Key ID, Secret Access Key, dan host endpoint ke `.env`.

Tidak ada konfigurasi CORS yang diperlukan: browser selalu berbicara ke origin
yang sama.

## Verifikasi

Statis: `go build ./...`, `go vet ./...`, `go test ./...`, `docker compose config`
tanpa warning, dan `nginx -t` atas template yang sudah dirender.

End-to-end terhadap bucket R2 sungguhan (dijalankan 2026-08-03, semua lolos):

| Langkah | Harapan |
|---|---|
| Startup backend | `Connected to R2 bucket: <bucket>` |
| Upload PDF | 200, `total_pages` dihitung dari file |
| Presigned URL | berprefix `/r2/`, bukan URL absolut R2 |
| GET penuh via nginx | 200, isi byte-identik dengan file asli |
| `Range: bytes=0-9` | 206 — jalur yang dipakai pdf.js |
| Akses tanpa token | 401 |
| DELETE ebook | 200, objek di R2 menjadi 404 |

## Catatan

Object key memakai UUID (`ebooks/<user-id>/<uuid>.pdf`), bukan nama file asli.
Karena itu tidak ada karakter khusus yang pernah masuk ke path presigned, dan
kekhawatiran encoding URI pada proxy nginx tidak berlaku dalam praktik.

Diagnosis dengan `curl --aws-sigv4` tidak dapat diandalkan untuk R2: curl salah
menandatangani request HEAD dan request ber-query-string, sehingga menghasilkan
401 palsu. Gunakan minio-go langsung saat memeriksa kredensial.
