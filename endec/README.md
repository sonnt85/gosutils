# endec

[![Go Reference](https://pkg.go.dev/badge/github.com/sonnt85/gosutils/endec.svg)](https://pkg.go.dev/github.com/sonnt85/gosutils/endec)

Encryption/decryption toolkit — AES-GCM and AES-CBC helpers, base64 utilities, hashing (MD5/SHA1), random number generation, 3DES, PKCS/PLP padding, and encrypted zip archives.

## Motivation

Go's `crypto/*` stdlib is powerful but low-level: to encrypt a byte slice with a passphrase you have to hand-roll cipher block, mode, nonce generation, and ciphertext layout every time. `endec` bundles the most common patterns into ready-to-use one-liners:

- **Passphrase-based encryption** without picking a KDF, mode, or nonce format — just call `EncrypBytes(data, passphrase)`.
- **Filename-embedded passphrases** — `DecryptFileWithPasswordInFileToBytes` parses `<name>s-nt<passphrase>` filenames, useful for shipping encrypted asset bundles where the key travels with the filename.
- **AES-encrypted zip archives** — combines `archive/zip` with per-chunk AES, so a whole directory becomes one encrypted file.
- **Utility grab-bag** — base64 with and without padding, random ints across sizes, PKCS/PLP padding, MD5/SHA1 hashing of files and strings.

## Installation

```bash
go get github.com/sonnt85/gosutils/endec
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/sonnt85/gosutils/endec"
)

func main() {
    plaintext := []byte("hello, world")
    passphrase := []byte("my-secret")

    ct, _ := endec.EncrypBytes(plaintext, passphrase)      // AES-GCM
    pt, _ := endec.DecryptBytes(ct, passphrase)
    fmt.Println(string(pt))                                 // "hello, world"

    // Base64-friendly variant (RawStdEncoding, no padding)
    ctstr, _ := endec.EncrypBytesToString(plaintext, passphrase)
    pt2, _ := endec.DecryptBytesFromString(ctstr, passphrase)
    fmt.Println(string(pt2))                                // "hello, world"
}
```

## Features

- **AES-GCM (authenticated)** and **AES-CBC (unauthenticated)** — byte, base64, and file variants
- Passphrase-based mode (auto key derivation via MD5) or raw-key mode (bring your own AES key)
- Encrypted zip archives with per-chunk AES stream
- Base64 encode/decode with or without padding
- MD5 / SHA1 hashing of bytes, strings, and files
- Cryptographically random integers across `int`, `int32`, `int64`, `uint`, `uint32`, `uint64`, and ranged variants
- PKCS5 / PKCS7 / PLP padding + unpadding
- 3DES CBC and ECB helpers
- Filename-embedded passphrase convention: `<name>s-nt<passphrase>` → `DecryptFileWithPasswordInFileToBytes`

## Usage

### Encrypted asset files with passphrase in filename

Used by `gstpl` for the `varenc.*` convention — assets are decrypted at startup by scanning for filenames matching the pattern.

```go
// A file at ./varenc.jsons-nt(my-passphrase) will be decrypted with passphrase "my-passphrase"
data, err := endec.DecryptFileWithPasswordInFileToBytes("./varenc.jsons-nt(my-passphrase)")
```

### File-to-file encryption

```go
_ = endec.EncryptFileToFile("./secret.txt", "./secret.enc", []byte("passphrase"))
_ = endec.DecryptFileToFile("./secret.enc", "./restored.txt", []byte("passphrase"))
```

### Raw-key AES-GCM (bring your own key)

```go
key := make([]byte, 32)                         // AES-256 key
_, _ = rand.Read(key)

ct, _ := endec.AESGCMEncrypt(key, plaintext)
pt, _ := endec.AESGCMDecrypt(key, ct)
```

### AES-CBC with explicit IV

```go
key := []byte("0123456789abcdef")                 // 16 bytes for AES-128
iv  := []byte("abcdef9876543210")                 // 16 bytes IV
ct, _ := endec.AESCBCEncrypt(key, plaintext, iv)
pt, _ := endec.AESCBCDecrypt(key, ct, iv)
```

### Encrypted zip directory

```go
_ = endec.ZipEncrypt("./data",   "./bundle.zip",   []byte("passphrase"))
_ = endec.ZipDecrypt("./bundle.zip", "./restored", false, "passphrase")
```

### Random IDs & hashing

```go
id  := endec.HexId(8)                              // 16-char hex random
n   := endec.RandRangeInt64(1000, 9999)            // random in [1000, 9999]
sum := endec.MD5(data)                             // hex string
sha := endec.SHA1(data)
```

## API Reference

Grouped by purpose. See `go doc github.com/sonnt85/gosutils/endec` for full signatures.

**Passphrase-based (recommended for most cases)**
- `EncrypBytes(data, passphrase) → ciphertext` / `DecryptBytes(ciphertext, passphrase) → plaintext` — AES-GCM byte-level.
- `EncrypBytesToString(...)` / `DecryptBytesFromString(...)` — same but Base64 (RawStdEncoding, no padding).
- `EncryptFileToBytes(filename, passphrase)` / `EncryptFileToFile(in, out, passphrase)` — file variants.
- `DecryptFileToBytes(filename_or_reader, passphrase)` / `DecryptFileToFile(in, out, passphrase)`.
- `DecryptFileWithPasswordInFileToBytes(pathfile)` — extract passphrase from filename `<name>s-nt<passphrase>`.
- `AesEncDecryptViaPassord(data, password, decrypt ...bool)` — dual-mode helper.

**Raw-key AES (bring your own key)**
- `AESGCMEncrypt(key, plaintext)` / `AESGCMDecrypt(key, ciphertext)` — authenticated.
- `AESGCMEncryptToBase64(...)` / `AESGCMDecryptFromBase64(...)` — base64 variants.
- `AESCBCEncrypt(key, plaintext, ivs...)` / `AESCBCDecrypt(key, ciphertext, ivs...)` — unauthenticated, needs explicit IV or generates one.
- `AESCBCEncryptToBase64(...)` / `AESCBCDecryptFromBase64(...)`.

**Encrypted zip archives**
- `ZipEncrypt(source, destination, password...)` / `ZipDecrypt(src, dstPath, removeSrc, passwords...)`.
- `ZipFile(source, destination, removeSrc)` — plain zip.
- `ZipOpen(src, password...)` → `*ZipReader` — random-access to encrypted archive.
- `NewAesDecrypter(password, chunkSize)` — plug into `archive/zip.RegisterDecompressor` for the custom AES codec (`ZIPTYPEAES = 0x66`).

**Base64 helpers**
- `Base64Encode(data) string` / `Base64Decode(string) ([]byte, error)` — standard padding.
- `Base64EncodeNoPadding(...)` / `Base64DecodeNoPadding(...)` — URL-safe or streaming.

**Hashing**
- `MD5(data) string` — hex of MD5.
- `MD5Bytes(data) []byte` — raw MD5.
- `MD5File(filename, chunkSizes...)`.
- `SHA1(data) string`.

**Random**
- `RandInt/RandInt32/RandInt64/RandUint/RandUint32/RandUint64` — full range, `crypto/rand`.
- `RandRangeInt64(from, to)` / `RandRangeInterger(from, to int)` — bounded.
- `HexId(n int) string` — random hex string of length `2*n`.

**Padding**
- `PKCS5Padding` / `PKCS5Unpadding`.
- `Pkcs7Padding` / `Pkcs7Unpadding`.
- `PLPPadding` / `PLPUnpadding`.

**3DES**
- `TripleDesEncrypt/TripleDesDecrypt` — CBC.
- `TripleEcbDesEncrypt/TripleEcbDesDecrypt` — ECB.

**String helpers**
- `StringSimpleEncrypt(input, key)` / `StringSimpleDecrypt(input, key)` — lightweight.
- `BytesZipToString(input)` / `StringUnzip(input)` — zip round-trip via string.

**Errors / constants**
- `ERR_CIPHERTEXT_TOO_SHORT` — returned by decrypt paths when input < nonce size.
- `ZIPTYPEAES = 0x66` — custom zip compression method ID.
- `AESZIPCHUNKSIZE` — global chunk size for `NewAesDecrypter`.

## Design Decisions & Trade-offs

**Key derivation is MD5(passphrase) → 16-byte AES-128 key.**
This is fast but **not a slow KDF** (unlike PBKDF2, scrypt, or Argon2). Any short or dictionary passphrase is offline-brute-forceable within minutes on modern hardware. Use `endec` only when:
- Passphrases are long and high-entropy (≥16 chars random), OR
- The threat model doesn't include offline attackers with the ciphertext, OR
- You wrap the caller with a proper KDF (`argon2.IDKey(...)`) and pass the derived key via the raw-key API (`AESGCMEncrypt(key, ...)`) instead.

**GCM by default (in `EncrypBytes` / `DecryptBytes`).**
Authenticated encryption. Tampering with ciphertext fails decryption cleanly, no silent corruption. Nonce is 12 bytes random per encryption, prepended to ciphertext: `nonce || ciphertext_with_tag`.

**CBC available for compatibility.**
`AESCBCEncrypt/Decrypt` exist for interop with systems that require CBC. **Not authenticated** — vulnerable to padding-oracle attacks if the caller reveals decryption errors. Prefer GCM.

**Base64 uses RawStdEncoding.**
No padding characters (`=`). Compatible with URL-safe contexts and slightly shorter output. Watch out: standard `base64.StdEncoding.DecodeString` will reject `endec`'s output.

## Concurrency & Thread-Safety

All package-level functions are stateless and safe for concurrent calls. `ZipReader` is not documented as concurrent-safe — assume single-goroutine use per instance.

## Gotchas

- **`ERR_CIPHERTEXT_TOO_SHORT`** — decrypting a slice smaller than the GCM nonce size (12 bytes) returns this. Guard length before calling on user input.
- **Nonce format is `nonce || ciphertext`.** Do not strip the leading 12 bytes accidentally when transporting.
- **`Base64Decode` vs `Base64DecodeNoPadding`** — must match the encode variant. Wrong pair returns "illegal base64 data".
- **`MD5` for hashing content is OK; `MD5Bytes(passphrase)` for KEY DERIVATION is weak.** See Design Trade-offs.
- **CBC IV**: `AESCBCEncrypt` with `ivs...` empty generates a random IV internally — but the API returns the ciphertext without the IV prepended (unlike GCM). Caller must track the IV separately. Prefer supplying IV explicitly.
- **`ZipEncrypt` with empty passphrase** — silently produces a plain zip (no encryption). Verify passphrase is non-empty in production paths.

## Ecosystem Usage

- **`gstpl`** uses `DecryptBytes` + the `varenc.<ext>s-nt<passphrase>` filename convention to decrypt embedded config assets at startup (see `internal/app/gvar.go` `Init()`).
- **`gsjson.Fjson`** and **`gsjson.EnvJson`** use `AESGCMEncrypt/Decrypt` internally for encrypted JSON storage.

## Author

**sonnt85** — [thanhson.rf@gmail.com](mailto:thanhson.rf@gmail.com)

## License

MIT.
