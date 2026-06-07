<div align="center">

<img src="web/static/img/logo.svg" width="64" height="64" alt="goform">

# goform

**Modern, sürükle-bırak, açık kaynak form oluşturucu — Go ile yazıldı, tek dosya, AI-dostu API.**

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://golang.org)
[![Docker](https://img.shields.io/badge/Docker-ready-2496ED?logo=docker&logoColor=white)](Dockerfile)

</div>

---

**goform**, Google Forms'a alternatif olarak tasarlanmış, modern ve hızlı bir form oluşturucudur. Sürükle-bırak editör, paylaşılabilir bağlantılar, yanıt analizi ve AI ajanları için tam kapsamlı bir REST API ile birlikte gelir. Tek bir Go binary'si olarak çalışır, harici bağımlılığı yoktur, ve verilerinizi SQLite'da saklar.

## ✨ Özellikler

- **Sürükle-bırak form editörü** — 14 farklı alan tipi (metin, e-posta, URL, telefon, çoktan seçmeli, onay kutusu, açılır menü, sayı, tarih, saat, yıldız puan, lineer ölçek, bölüm başlığı)
- **6 hazır şablon** — Müşteri memnuniyet, etkinlik kayıt, iş başvurusu, iletişim, geri bildirim formları ve boş şablon
- **Form teması** — Her form için 6 renk paleti, 16 emoji ikon, ve **açık/koyu/otomatik** mod seçenekleri
- **Yanıt limiti** — Form başına maksimum yanıt sayısı; limite ulaşılınca otomatik kapanır ve özelleştirilebilir mesaj gösterir
- **Webhook bildirimleri** — Yanıt geldiğinde otomatik **Discord**, **Telegram**, **SMTP e-posta** bildirimleri
- **Çok kullanıcılı sistem** — İlk kayıt olan **süper admin**, sonra admin diğer kullanıcıları ekler
- **AI ajanları için tam API** — `/api/help` ile makine-okunabilir endpoint metadatası, `Bearer` token ile her şey otomasyona açık
- **Otomatik kayıt** — Yazdıkça arka planda kaydedilir
- **Yanıt analizi** — Anketler için ortalama/dağılım grafikleri, CSV indirme
- **Modern tema** — Açık/koyu/sistem teması (üst menüden hızlı geçiş), mobil uyumlu, klavye kısayolları
- **Şifre göster/gizle** — Tüm şifre alanlarında göz ikonu
- **Güvenlik** — bcrypt (cost 12), oturum çerezi (HTTP-only + SameSite), token rate limiting, CSP header'ları
- **Tek binary** — Tüm CSS/JS/HTML embed edilmiş, dış bağımlılık yok
- **Düşük kaynak** — Çalışırken ~13 MB RSS bellek kullanır

## 🚀 Hızlı Başlangıç

### Docker (önerilen)

```bash
docker run -d --name goform -p 3000:3000 -v goform-data:/data ghcr.io/benyigiteren/goform:latest
```

Sonra tarayıcıdan `http://localhost:3000` adresine gidin — ilk kayıt olan kişi süper admin olur.

### Docker Compose

```bash
git clone https://github.com/benyigiteren/goform.git
cd goform
docker compose up -d
```

### Yerel olarak (Go gerektirir)

```bash
git clone https://github.com/benyigiteren/goform.git
cd goform
go run .
```

İlk açılışta `http://localhost:3000/setup` sizi karşılar.

### Önceden derlenmiş binary

```bash
go install github.com/benyigiteren/goform@latest
goform --addr :3000 --data ./data
```

## ⚙️ Yapılandırma

| Değişken / Bayrak | Varsayılan | Açıklama |
|---|---|---|
| `--addr` / `GOFORM_ADDR` | `:3000` | Dinleme adresi |
| `--data` / `GOFORM_DATA` | `./data` | SQLite veritabanı dizini |
| `--public-url` / `GOFORM_PUBLIC_URL` | _(istek host'undan)_ | Paylaşım bağlantıları ve API örneklerinde kullanılacak public URL |

Üretimde Caddy/Nginx gibi bir reverse proxy arkasında HTTPS ile çalıştırın. goform `X-Forwarded-Proto`, `X-Forwarded-Host` ve `X-Forwarded-For` başlıklarını tanır.

## 🔔 Webhook bildirimleri

Her form için ayrı ayrı bildirim kanalları yapılandırılabilir. Builder sayfasında üstteki çan ikonuna tıklayın:

### Discord
Sunucu Ayarları → Entegrasyonlar → Webhook'lar → Yeni webhook oluştur → URL kopyala. URL'i `discord.webhookUrl` alanına yapıştırın.

### Telegram
1. [@BotFather](https://t.me/BotFather) ile yeni bot oluşturun (`/newbot`) — bot token alın
2. Botu kendi hesabınızla konuşturun veya gruba ekleyin
3. Chat ID için [@userinfobot](https://t.me/userinfobot) kullanabilirsiniz; grup için bot'u admin yapıp grubunuza eklemeniz gerekir

### SMTP (e-posta)
Gmail örneği:
- `smtpHost`: `smtp.gmail.com`
- `smtpPort`: `587`
- `smtpUser`: Gmail adresiniz
- `smtpPass`: [App password](https://support.google.com/accounts/answer/185833) (2FA aktif olmalı)
- `from`: Gönderici e-posta
- `to`: Alıcı(lar) — virgülle ayrılmış birden fazla adres

Builder'daki **Test bildirimi gönder** butonu ile kurulumunuzu doğrulayabilirsiniz.

## 🔑 Kimlik Doğrulama

İki tür kimlik doğrulama desteklenir:

### 1. Oturum çerezi (tarayıcı kullanımı için)

Login endpoint'i HTTP-only çerez döner; sonraki tüm istekler otomatik olarak kimlik doğrulamasıyla yapılır.

### 2. Bearer token (AI ajanları / otomasyonlar için)

`Ayarlar → API Erişimi` sayfasından bir token oluşturun. Token sadece bir kez gösterilir, kaybederseniz yenisini oluşturmalısınız.

```bash
curl https://goform.example.com/api/forms \
  -H "Authorization: Bearer gft_xxxxxxxxxxxx"
```

## 📡 REST API

Tüm endpoint'ler JSON kabul eder ve döner. Tarih alanları **Unix timestamp** (saniye) cinsindendir.

### Auth

| Method | Path | Auth | Açıklama |
|---|---|---|---|
| POST   | `/api/auth/setup`           | yok      | İlk kurulum: süper admin oluşturur |
| POST   | `/api/auth/login`           | yok      | E-posta + şifre ile giriş |
| POST   | `/api/auth/logout`          | yok      | Çıkış (oturumu siler) |
| GET    | `/api/auth/me`              | gerekli  | Mevcut kullanıcı bilgileri |
| POST   | `/api/auth/change-password` | gerekli  | Kendi şifrenizi değiştir |

### Formlar

| Method | Path | Auth | Açıklama |
|---|---|---|---|
| GET    | `/api/forms`           | gerekli | Formlarınız (admin: `?scope=all` ile tümü) |
| POST   | `/api/forms`           | gerekli | Yeni form oluştur |
| GET    | `/api/forms/:id`       | sahip/admin | Form detayı |
| PUT    | `/api/forms/:id`       | sahip/admin | Form güncelle |
| DELETE | `/api/forms/:id`       | sahip/admin | Form sil |

**Form payload alanları:**

| Alan | Tip | Açıklama |
|---|---|---|
| `title` | string | Form başlığı |
| `description` | string | Açıklama |
| `fields` | `Field[]` | Soru listesi |
| `theme` | `{color, icon, mode}` | Renk (`indigo`/`rose`/`emerald`/`amber`/`sky`/`slate`), emoji ikon, mod (`light`/`dark`/`auto`) |
| `accepting` | bool | Yanıt kabul ediyor mu |
| `maxResponses` | int / null | Maksimum yanıt sayısı (boş = sınırsız) |
| `closedMessage` | string | Form kapandığında / dolu olduğunda gösterilen özel mesaj |

### Bildirimler (sahip/admin)

| Method | Path | Açıklama |
|---|---|---|
| GET    | `/api/forms/:id/notifications`       | Form bildirim ayarlarını oku |
| PUT    | `/api/forms/:id/notifications`       | Discord / Telegram / SMTP yapılandırması |
| POST   | `/api/forms/:id/notifications/test`  | Yapılandırmayı test et |

### Herkese açık form (auth gerekmez)

| Method | Path | Açıklama |
|---|---|---|
| GET    | `/api/public/forms/:id`            | Form alanlarını oku (doldurmak için) |
| POST   | `/api/public/forms/:id/responses`  | Yanıt gönder |

### Yanıtlar

| Method | Path | Auth | Açıklama |
|---|---|---|---|
| GET    | `/api/forms/:id/responses` | sahip/admin | Form yanıtları |
| DELETE | `/api/responses/:id`       | sahip/admin | Yanıt sil |

### API Tokenları

| Method | Path | Auth | Açıklama |
|---|---|---|---|
| GET    | `/api/tokens`     | gerekli | Kendi tokenlarınız |
| POST   | `/api/tokens`     | gerekli | Yeni token (oluşturulan token yalnız bir kez döner) |
| DELETE | `/api/tokens/:id` | gerekli | Token iptal et |

### Kullanıcılar (yalnız admin)

| Method | Path | Açıklama |
|---|---|---|
| GET    | `/api/users`                     | Kullanıcı listesi |
| POST   | `/api/users`                     | Yeni kullanıcı oluştur |
| PUT    | `/api/users/:id`                 | Ad / rol güncelle |
| POST   | `/api/users/:id/reset-password`  | Kullanıcının şifresini sıfırla |
| DELETE | `/api/users/:id`                 | Kullanıcıyı sil |

### Diğer

| Method | Path | Açıklama |
|---|---|---|
| GET    | `/api/status` | Sürüm, kurulum durumu, public URL |
| GET    | `/api/help`   | **Makine-okunabilir** tüm endpoint listesi (AI ajanları için) |

## 🤖 AI Agent Entegrasyonu

`/api/help` endpoint'i auth gerektirmez ve sistemdeki tüm endpoint'leri JSON formatında döner. Bir AI agent buradan API yapısını öğrenebilir:

```bash
curl https://goform.example.com/api/help | jq
```

Token oluşturulduktan sonra her şey API üzerinden yapılabilir:

```bash
TOKEN="gft_xxxxxxxxxxxx"
BASE="https://goform.example.com"

# 1) Yeni form oluştur (yanıt limiti + tema dahil)
FORM_ID=$(curl -s $BASE/api/forms \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Etkinlik kaydı",
    "description": "İlk 50 kişi katılabilir",
    "theme": {"color": "rose", "icon": "🎉", "mode": "light"},
    "maxResponses": 50,
    "closedMessage": "Etkinlik için yer kalmadı. Bizi takip etmeye devam edin!",
    "fields": [
      {"id":"a1","type":"short_text","label":"Adınız","required":true},
      {"id":"a2","type":"email","label":"E-posta","required":true},
      {"id":"a3","type":"radio","label":"Katılım","required":true,"options":["Geliyorum","Gelmiyorum"]}
    ]
  }' | jq -r '.id')

echo "Form: $BASE/f/$FORM_ID"

# 2) Discord webhook bağla
curl -s $BASE/api/forms/$FORM_ID/notifications \
  -X PUT \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"discord":{"enabled":true,"webhookUrl":"https://discord.com/api/webhooks/..."}}'

# 3) Yanıtları çek
curl -s $BASE/api/forms/$FORM_ID/responses \
  -H "Authorization: Bearer $TOKEN" | jq
```

Python örneği:

```python
import requests

API = "https://goform.example.com/api"
TOKEN = "gft_xxxxxxxxxxxx"
H = {"Authorization": f"Bearer {TOKEN}"}

# Form oluştur
r = requests.post(f"{API}/forms", headers=H, json={
    "title": "AI tarafından oluşturuldu",
    "fields": [
        {"id": "f1", "type": "short_text", "label": "Adınız", "required": True},
        {"id": "f2", "type": "rating", "label": "Puan", "maxStars": 5},
    ],
})
form_id = r.json()["id"]

# Yanıtları analiz et
responses = requests.get(f"{API}/forms/{form_id}/responses", headers=H).json()
avg = sum(r["data"]["f2"] for r in responses) / len(responses) if responses else 0
print(f"Ortalama puan: {avg:.2f} / {len(responses)} yanıt")
```

### Hata kodları

Public form endpoint'leri yapılandırılmış hata kodları döner — AI agent farklı durumları ayırt edebilir:

| Code | HTTP | Anlamı |
|---|---|---|
| `not_found` | 404 | Form mevcut değil |
| `form_closed` | 403 | Form sahibi durdurmuş; özel mesaj `error` alanında |
| `max_reached` | 403 | Maksimum yanıt sayısına ulaşıldı; `formTitle`, `maxResponses`, `responseCount` döner |

## 🧱 Alan Tipleri

| Tip | Açıklama | Ek alanlar |
|---|---|---|
| `short_text` | Tek satır metin | `placeholder` |
| `long_text`  | Çok satır metin | `placeholder` |
| `email`      | E-posta (otomatik doğrulama) | `placeholder` |
| `url`        | Web adresi (otomatik doğrulama) | `placeholder` |
| `phone`      | Telefon numarası | `placeholder` |
| `number`     | Sayı | `min`, `max`, `placeholder` |
| `date`       | Tarih |  |
| `time`       | Saat |  |
| `radio`      | Tek seçimli liste | `options: []` |
| `checkbox`   | Çok seçimli liste | `options: []` |
| `dropdown`   | Açılır menü | `options: []` |
| `rating`     | Yıldız puanlama | `maxStars` (3-10) |
| `scale`      | Lineer ölçek | `min`, `max`, `minLabel`, `maxLabel` |
| `section`    | Bölüm başlığı (yanıt almaz) | `description` |

Tüm alanlar `required` (zorunlu) olarak işaretlenebilir.

## 🛡️ Güvenlik

- **Şifreler**: bcrypt cost 12 ile hash'lenir
- **Oturumlar**: 256-bit rasgele token, HTTP-only + SameSite=Lax çerez, 30 gün ömür
- **API tokenları**: SHA-256 hash olarak saklanır, asla düz metin olarak gösterilmez (sadece prefix görüntülenir)
- **Rate limiting**: IP başına 15 dakikada en fazla 10 başarısız giriş denemesi
- **Güvenlik başlıkları**: `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, sıkı CSP
- **SQL injection**: Sadece prepared statement kullanılır
- **XSS**: Tüm kullanıcı girdileri kaçırılır
- **Şifre değişimi**: Tüm diğer oturumları sonlandırır
- **HTTPS**: Reverse proxy arkasında çalıştırın

### Güvenlik açığı bildirme

Güvenlik açıklarını lütfen `security@example.com` adresine bildirin, public issue olarak açmayın.

## 🏗️ Mimari

```
goform/
├── main.go         # giriş, embed FS, HTTP server
├── db.go           # SQLite şema, CRUD
├── auth.go         # bcrypt, oturum, middleware
├── handlers.go     # HTTP rota'ları ve handler'lar
├── web/
│   ├── static/     # CSS, JS, logo (embed)
│   └── pages/      # HTML şablonları (embed)
├── Dockerfile      # Multi-stage, distroless final
└── docker-compose.yml
```

### Teknoloji

- **Backend**: Go 1.22+ (`net/http` ServeMux pattern matching)
- **DB**: SQLite via `modernc.org/sqlite` (saf Go, CGO yok)
- **Hash**: `golang.org/x/crypto/bcrypt`
- **Frontend**: Vanilla HTML/CSS/JS — kütüphane yok
- **Embed**: Tüm statikler `//go:embed` ile binary'ye gömülü

### Bellek kullanımı

Linux/Docker üzerinde tipik kullanım: **~12-15 MB RSS** (yerleşik bellek), kullanıcı sayısı ve trafik ile yumuşak şekilde büyür.

## 🤝 Katkı

Issue ve PR'lar memnuniyetle karşılanır. Büyük değişiklikler öncesi bir issue açmanız önerilir.

```bash
git clone https://github.com/benyigiteren/goform.git
cd goform
go run .
```

## 📄 Lisans

[MIT](LICENSE) — özgürce kullanın, değiştirin, dağıtın.

---

<div align="center">

Made with ❤️ in Türkiye · [Issues](https://github.com/benyigiteren/goform/issues) · [Discussions](https://github.com/benyigiteren/goform/discussions)

</div>
