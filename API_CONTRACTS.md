# SmartKlap — Mobile API Contracts

این سند بر اساس بررسی کامل کد فرانت (`src/app/**`) نوشته شده: هر endpoint دقیقاً همون فیلدهایی رو دارد که UI فعلی (با mock data) نمایش می‌دهد یا جمع‌آوری می‌کند. هدف این سند: تحویل به بک‌اند‌نویس برای پیاده‌سازی. فرمت هر Screen طبق تمپلیت "Mobile API Contract" است.

**قراردادهای مشترک (برای همه endpoint ها یکسان است، تکرار نشده):**

- **Base URL:** `{API_URL}` (فعلاً placeholder در `src/constants/config.ts`, باید مقدار واقعی گرفته و در `.env` گذاشته شود)
- **Prefix:** `/api/v1`
- **Content-Type:** `application/json` مگر جایی که multipart مشخص شده (آپلود عکس/ویدیو)
- **Authentication:** `Authorization: Bearer <access_token>` — از `src/lib/axios.ts` خودکار اضافه می‌شود؛ توکن بعد از verify OTP گرفته و در MMKV (`token`) ذخیره می‌شود؛ پاسخ `401` باعث حذف توکن سمت کلاینت و ریدایرکت به Sign Up می‌شود
- **کلیدهای JSON:** `snake_case`
- **کدهای خطای پیش‌فرض** (مگر برای هر endpoint موارد اضافه ذکر شده باشد):
  - `400` Validation error
  - `401` توکن نامعتبر/منقضی
  - `403` عدم دسترسی
  - `404` منبع پیدا نشد
  - `422` خطای منطق بیزینس (مثلاً quiz قبلاً جواب داده شده)
  - `429` تعداد درخواست زیاد
  - `500` خطای سرور
- **Pagination:** برای لیست‌ها `?page=1&limit=20` + پاسخ شامل `"meta": { "page", "limit", "total", "total_pages" }`؛ برای endpoint های aggregate/dashboard (مثل Home) pagination نداریم — فقط preview با تعداد ثابت برمی‌گردد.

> ⚠️ **نکته مهم برای بک‌اند‌نویس:** در حال حاضر `src/api/auth.ts` با `login(email, password)` نوشته شده، ولی UI فعلی (`LoginScreen`, `SignUpScreen`, `VerifyCodeScreen`) کاملاً **passwordless / OTP-based** است (فقط name+email می‌گیرد، بعد کد ۴ رقمی تایید می‌شود). این سند بر اساس UI واقعی (OTP) نوشته شده، نه فایل فعلی `auth.ts`.

---

## فهرست

1. [Auth](#1-auth-module)
2. [Profile](#2-profile-module)
3. [Home](#3-home-module)
4. [Chants](#4-chants-module)
5. [Guess](#5-guess-module)
6. [Snacks (Food ordering)](#6-snacks-module)
7. [Store (Merch)](#7-store-module)
8. [Video](#8-video-module)
9. [Statistics](#9-statistics-module)

---

## 1. Auth Module

### 1.1 Sign Up Screen

| Field | Value |
|---|---|
| **Screen Name** | Sign Up Screen |
| **Endpoint** | `/api/v1/auth/register` |
| **HTTP Method** | POST |
| **Request** | `{ "name": "string", "email": "string" }` |
| **Response** | `200 OK` — `{ "otp_sent": true }` (کاربر هنوز در DB ذخیره نمی‌شود؛ بعد از `verify-otp` ساخته می‌شود) |
| **Authentication** | None |
| **Error Codes** | `400` نام/ایمیل نامعتبر · `409` ایمیل قبلاً ثبت شده · `429` cooldown ارسال مجدد · `500` |
| **Pagination** | ندارد |

```json
{ "otp_sent": true }
```

> Resend ثبت‌نام: دوباره همین `POST /auth/register` را بزنید (نه login).

### 1.2 Verify Code Screen

| Field | Value |
|---|---|
| **Screen Name** | Verify Code Screen |
| **Endpoint** | `/api/v1/auth/verify-otp` |
| **HTTP Method** | POST |
| **Request** | `{ "email": "string", "code": "string(4)" }` |
| **Response** | `200 OK` — `{ "access_token", "refresh_token" }` |
| **Authentication** | None (نیاز به `email` از مرحله قبل) |
| **Error Codes** | `400` · `401` کد نامعتبر/منقضی · `429` تعداد تلاش زیاد · `500` |
| **Pagination** | ندارد |

Resend: endpoint جدا ندارد — دوباره `POST /api/v1/auth/login` (یا برای کاربر جدید بعد از ثبت‌نام همان login) را صدا بزنید.

```json
{
  "access_token": "eyJhbGciOi...",
  "refresh_token": "eyJhbGciOi..."
}
```


### 1.3 Login Screen

| Field | Value |
|---|---|
| **Screen Name** | Login Screen |
| **Endpoint** | `/api/v1/auth/login` |
| **HTTP Method** | POST |
| **Request** | `{ "email": "string" }` |
| **Response** | `200 OK` — `{ "otp_sent": true }` (کد به همان `verify-otp` بالا ارسال و تایید می‌شود) |
| **Authentication** | None |
| **Error Codes** | `404` ایمیل ثبت‌نشده · `429` · `500` |
| **Pagination** | ندارد |

```json
{ "otp_sent": true }
```

---

## 2. Profile Module

### 2.1 Profile Screen

| Field | Value |
|---|---|
| **Screen Name** | Profile Screen |
| **Endpoint** | `/api/v1/profile/me` |
| **HTTP Method** | GET |
| **Request** | بدون پارامتر |
| **Response** | `200 OK` — `{ "id", "created_at", "updated_at", "name", "email", "avatar_url", "points" }` + لیدربرد جدا |
| **Authentication** | Bearer |
| **Error Codes** | استاندارد |
| **Pagination** | ندارد |

**لیدربرد (کارت‌های "Top ranks"):** `GET /api/v1/profile/leaderboard?limit=4`

```json
{
  "name": "John Smith",
  "avatar_url": "https://cdn.smartklap.com/avatars/usr_8f21.png",
  "points": 960,
  "rank": { "position": 3, "total": 128 }
}
```

```json
{
  "items": [
    { "rank": 1, "name": "Bahman", "points": 690, "avatar_url": "https://cdn.smartklap.com/avatars/u1.png" },
    { "rank": 2, "name": "alireza", "points": 640, "avatar_url": "https://cdn.smartklap.com/avatars/u2.png" }
  ]
}
```

### 2.2 Edit Profile Screen

| Field | Value |
|---|---|
| **Screen Name** | Edit Profile Screen |
| **Endpoint** | `/api/v1/profile/me` |
| **HTTP Method** | PATCH |
| **Request** | `{ "name"?: "string", "email"?: "string" }` |
| **Response** | `200 OK` — same shape as `GET /profile/me` |
| **Authentication** | Bearer |
| **Error Codes** | `400` / `422` ایمیل تکراری · استاندارد |
| **Pagination** | ندارد |

**آپلود آواتار (جدا، multipart):** `POST /api/v1/profile/me/avatar` — فیلد `avatar` (image, max ۵MB) → `{ "avatar_url": "string" }`. خطای اضافه: `413` حجم فایل زیاد.

```json
{ "name": "John Smith", "email": "john@example.com", "avatar_url": "https://cdn.smartklap.com/avatars/usr_8f21.png", "points": 960, "rank": { "position": 3, "total": 128 } }
```

---

## 3. Home Module

اپ دو حالت Home دارد: **Stadium Mode** (فعال، تب مرکزی) و **Club Mode** (کد آماده است ولی به هیچ route وصل نیست — `SoccerClubScreen.tsx`). هر دو را جدا تعریف می‌کنم چون داده‌شان کاملاً متفاوت است.

### 3.1 Home Screen — Stadium Mode

| Field | Value |
|---|---|
| **Screen Name** | Home Screen (Stadium Mode) |
| **Endpoint** | `/api/v1/mobile/home/stadium` |
| **HTTP Method** | GET |
| **Request** | بدون پارامتر (هدر Auth کافیست) |
| **Response** | `200 OK` — `user_summary`, `live_match`, `chant_program`, `foods[]` |
| **Authentication** | Bearer |
| **Error Codes** | استاندارد + `404` اگر مسابقه لایو وجود ندارد (فیلد `live_match` را `null` برگردانید، نه ۴۰۴) |
| **Pagination** | ندارد — `foods` فقط ۳ آیتم preview؛ لیست کامل از `GET /api/v1/snacks` |

```json
{
  "user_summary": { "points": 289, "cart_count": 1 },
  "live_match": {
    "id": "m_1023",
    "status": "live",
    "minute": "34:18",
    "home_team": { "name": "SP Burgos", "logo_url": "https://cdn.smartklap.com/logos/burgos.png", "score": 3 },
    "away_team": { "name": "FC Barcelona", "logo_url": "https://cdn.smartklap.com/logos/barcelona.png", "score": 2 }
  },
  "chant_program": {
    "today_points": 127,
    "today_target": 500,
    "recent_items": [
      { "id": "cp_1", "title": "Chant number 1", "subtitle": "You've gotten 100 point", "minutes_ago": 10, "status": "done" }
    ]
  },
  "foods": [
    { "id": "f_1", "name": "Double Berger", "description": "With onions", "price": "8,20 €", "image_url": "https://cdn.smartklap.com/foods/berger.png" }
  ]
}
```

### 3.2 Home Screen — Club Mode

| Field | Value |
|---|---|
| **Screen Name** | Home Screen (Club Mode) |
| **Endpoint** | `/api/v1/mobile/home/club` |
| **HTTP Method** | GET |
| **Request** | بدون پارامتر |
| **Response** | `200 OK` — `upcoming_matches[]`, `club_store[]` |
| **Authentication** | Bearer |
| **Error Codes** | استاندارد |
| **Pagination** | ندارد — هر دو preview هستند؛ لیست کامل: `GET /api/v1/matches`, `GET /api/v1/products` |

```json
{
  "upcoming_matches": [
    {
      "id": "um_1",
      "home_name": "SP Burgos",
      "home_logo_url": "https://cdn.smartklap.com/logos/burgos.png",
      "away_name": "FC Barcelona",
      "away_logo_url": "https://cdn.smartklap.com/logos/barcelona.png",
      "date": "2026-03-01",
      "time": "18:30",
      "status": "upcoming",
      "countdown_seconds": 128900,
      "score": null
    }
  ],
  "club_store": [
    { "id": "s_1", "name": "Sport T-shirt", "price": "32.50 €", "image_url": "https://cdn.smartklap.com/store/shirt.png" }
  ]
}
```

> `status` یکی از `upcoming` / `live` / `finished` است (معادل `hasDetails`/`upcomming`/`finished` فعلی در کد؛ پیشنهاد می‌کنم این سه اسم استاندارد را در بک‌اند استفاده کنید). موقع `finished`، `score` پر و `countdown_seconds` صفر است.

---

## 4. Chants Module

### 4.1 Chant Screen (list)

| Field | Value |
|---|---|
| **Screen Name** | Chant Screen |
| **Endpoint** | `/api/v1/chants` |
| **HTTP Method** | GET |
| **Request** | Query: `search?: string`, `match_id?: string` |
| **Response** | `200 OK` — `{ "match_title", "sections": [{ "title", "items": [...] }] }` |
| **Authentication** | Bearer |
| **Error Codes** | استاندارد |
| **Pagination** | ندارد (لیست بر اساس section گروه‌بندی می‌شود؛ هر section کوتاه است) |

```json
{
  "match_title": "SP Burgos - FC Barcelona's chants",
  "sections": [
    {
      "title": "Todays chants",
      "items": [
        { "id": 1, "title": "We will rock you", "song_points": 200, "duration_seconds": 120, "is_done": true, "is_next": false, "is_liked": false, "is_preview": false }
      ]
    }
  ]
}
```

### 4.2 Count Down Screen

| Field | Value |
|---|---|
| **Screen Name** | Count Down Screen |
| **Endpoint** | `/api/v1/chants/{chant_id}/countdown` |
| **HTTP Method** | GET |
| **Request** | Path param `chant_id` |
| **Response** | `200 OK` — `{ "title", "points", "today_points", "today_target", "countdown_seconds" }` |
| **Authentication** | Bearer |
| **Error Codes** | `404` chant پیدا نشد · استاندارد |
| **Pagination** | ندارد |

```json
{ "title": "We will rock you", "points": 34, "today_points": 127, "today_target": 500, "countdown_seconds": 90 }
```

### 4.3 Chant Lyrics Screen

| Field | Value |
|---|---|
| **Screen Name** | Chant Lyrics Screen |
| **Endpoint** | `/api/v1/chants/{chant_id}/lyrics` |
| **HTTP Method** | GET |
| **Request** | Path param `chant_id`; query `mode: "preview" \| "main"` (فقط برای لاگ سمت بک‌اند، تاثیری در پاسخ ندارد) |
| **Response** | `200 OK` — `{ "title", "audio_url", "lyrics": [{ "id", "time_seconds", "text", "flash_duration_ms", "vibration_duration_ms" }] }` |
| **Authentication** | Bearer |
| **Error Codes** | `404` · استاندارد |
| **Pagination** | ندارد |

**تکمیل چنت:** `POST /api/v1/chants/{chant_id}/complete` → `{ "points_earned": 100, "total_points": 227 }`

```json
{
  "title": "We will rock you",
  "audio_url": "https://cdn.smartklap.com/audio/chant_1.mp3",
  "lyrics": [
    { "id": 1, "time_seconds": 0, "text": "Buddy you're a boy make a big noise", "flash_duration_ms": 500, "vibration_duration_ms": 500 },
    { "id": 2, "time_seconds": 4, "text": "Playin' in the street gonna be a big man some day", "flash_duration_ms": 300, "vibration_duration_ms": 300 }
  ]
}
```

---

## 5. Guess Module

### 5.1 Guess Match Screen

| Field | Value |
|---|---|
| **Screen Name** | Guess Match Screen |
| **Endpoint** | `/api/v1/guess/matches/{match_id}` |
| **HTTP Method** | GET |
| **Request** | Path param `match_id` (یا `current` برای مسابقه فعال) |
| **Response** | `200 OK` — `{ "match": {...}, "participation_points": 100, "quizzes": [{ "id", "title", "points", "is_done" }] }` |
| **Authentication** | Bearer |
| **Error Codes** | `404` مسابقه پیدا نشد · استاندارد |
| **Pagination** | ندارد |

```json
{
  "match": {
    "home_name": "SP Burgos", "home_role": "Home", "home_logo_url": "https://cdn.smartklap.com/logos/burgos.png",
    "away_name": "FC Barcelona", "away_role": "Away", "away_logo_url": "https://cdn.smartklap.com/logos/barcelona.png",
    "competition_logo_url": "https://cdn.smartklap.com/logos/uefa.png",
    "date": "2026-03-01", "time": "18:30"
  },
  "participation_points": 100,
  "quizzes": [
    { "id": "q_1", "title": "Result of the game", "points": 600, "is_done": true },
    { "id": "q_2", "title": "Best Player", "points": 200, "is_done": false }
  ]
}
```

### 5.2 Result Of Game Screen (quiz answer)

| Field | Value |
|---|---|
| **Screen Name** | Result Of Game Screen |
| **Endpoint** | `/api/v1/guess/quizzes/{quiz_id}/answer` |
| **HTTP Method** | POST |
| **Request** | `{ "choice": "barcelona" \| "burgos" \| "draw" }` (برای quiz های دیگر مثل Best Player، `choice` می‌تواند `player_id` باشد — نوع quiz را در `GET /guess/quizzes/{id}` مشخص کنید) |
| **Response** | `200 OK` — `{ "status": "submitted", "points_earned": 100 }` (امتیاز نهایی صحیح بعد پایان مسابقه از طریق push/poll اعلام می‌شود) |
| **Authentication** | Bearer |
| **Error Codes** | `409` قبلاً جواب داده شده · `422` مسابقه شروع/تمام شده · استاندارد |
| **Pagination** | ندارد |

سوالات هر quiz: `GET /api/v1/guess/quizzes/{quiz_id}` → `{ "id", "title", "options": [{ "id", "label" }] }`

```json
{ "status": "submitted", "points_earned": 100 }
```

---

## 6. Snacks Module (Food ordering)

### 6.1 Snacks Screen

| Field | Value |
|---|---|
| **Screen Name** | Snacks Screen |
| **Endpoint** | `/api/v1/snacks` |
| **HTTP Method** | GET |
| **Request** | Query: `search?`, `category?: "sandwiches"\|"snacks"\|"drinks"`, `currency?: "EUR"\|"POINT"`, `page?`, `limit?` |
| **Response** | `200 OK` — `{ "items": [{ "id","name","description","price","image_url" }], "cart_count", "meta" }` |
| **Authentication** | Bearer |
| **Error Codes** | استاندارد |
| **Pagination** | بله — `page`/`limit`/`meta` |

```json
{
  "items": [
    { "id": "1", "name": "Double Berger", "description": "With onions", "price": "8,20 €", "image_url": "https://cdn.smartklap.com/foods/berger.png" }
  ],
  "cart_count": 2,
  "meta": { "page": 1, "limit": 20, "total": 10, "total_pages": 1 }
}
```

### 6.2 Snack Detail Screen

| Field | Value |
|---|---|
| **Screen Name** | Snack Detail Screen |
| **Endpoint** | `/api/v1/snacks/{snack_id}` |
| **HTTP Method** | GET |
| **Request** | Path param `snack_id` |
| **Response** | `200 OK` — `{ "id","name","description","price","image_url" }` |
| **Authentication** | Bearer |
| **Error Codes** | `404` · استاندارد |
| **Pagination** | ندارد |

```json
{ "id": "1", "name": "Double Berger", "description": "With onions", "price": "44,00 €", "image_url": "https://cdn.smartklap.com/foods/berger.png" }
```

### 6.3 Basket Screen

| Field | Value |
|---|---|
| **Screen Name** | Basket Screen |
| **Endpoint** | `/api/v1/cart` |
| **HTTP Method** | GET |
| **Request** | بدون پارامتر |
| **Response** | `200 OK` — `{ "orders": [{ "id","title","date","items":[{ "id","image_url","quantity" }],"extra_text" }] }` |
| **Authentication** | Bearer |
| **Error Codes** | استاندارد |
| **Pagination** | ندارد (سبد فعلی کاربر) |

**مدیریت سبد:**
- `POST /api/v1/cart/items` — `{ "product_type": "snack"\|"merch", "product_id", "quantity" }`
- `PATCH /api/v1/cart/items/{item_id}` — `{ "quantity" }`
- `DELETE /api/v1/cart/items/{item_id}`

```json
{
  "orders": [
    {
      "id": "food_1",
      "title": "Food Delivery",
      "date": "2026-07-10",
      "items": [
        { "id": "f1", "image_url": "https://cdn.smartklap.com/foods/berger.png", "quantity": 2 }
      ],
      "extra_text": "2 more item"
    }
  ]
}
```

### 6.4 Checkout Screen

| Field | Value |
|---|---|
| **Screen Name** | Checkout Screen |
| **Endpoint** | `/api/v1/orders` |
| **HTTP Method** | POST |
| **Request** | `{ "delivery_method": "seat" \| "pickup", "seat_number"?: "string" }` (روی سبد فعلی کاربر عمل می‌کند) |
| **Response** | `201 Created` — `{ "order_id","items":[...],"subtotal","shipping","total","status": "pending_payment" }` |
| **Authentication** | Bearer |
| **Error Codes** | `422` سبد خالی · استاندارد |
| **Pagination** | ندارد |

پرداخت جدا: `POST /api/v1/orders/{order_id}/pay` — `{ "payment_method" }` → `{ "status": "paid" }`

```json
{
  "order_id": "ord_501",
  "items": [
    { "id": "1", "name": "Double Berger", "quantity": 1, "price": "8,20 €" }
  ],
  "subtotal": "40,00 €",
  "shipping": "4,00 €",
  "total": "44,00 €",
  "status": "pending_payment"
}
```

---

## 7. Store Module (Merch)

### 7.1 Store Screen

| Field | Value |
|---|---|
| **Screen Name** | Store Screen |
| **Endpoint** | `/api/v1/products` |
| **HTTP Method** | GET |
| **Request** | Query: `search?`, `category?: "t-shirts"\|"balls"\|"stickers"\|"sport-suits"`, `currency?: "EUR"\|"POINT"`, `page?`, `limit?` |
| **Response** | `200 OK` — `{ "items": [{ "id","name","description","price","image_url" }], "cart_count", "meta" }` |
| **Authentication** | Bearer |
| **Error Codes** | استاندارد |
| **Pagination** | بله |

```json
{
  "items": [
    { "id": "1", "name": "Sport T-shirt", "description": "Home kit", "price": "32,50 €", "image_url": "https://cdn.smartklap.com/store/shirt.png" }
  ],
  "cart_count": 2,
  "meta": { "page": 1, "limit": 20, "total": 10, "total_pages": 1 }
}
```

### 7.2 Product Detail Screen

| Field | Value |
|---|---|
| **Screen Name** | Product Detail Screen |
| **Endpoint** | `/api/v1/products/{product_id}` |
| **HTTP Method** | GET |
| **Request** | Path param `product_id`; query `size?: "M"\|"L"\|"XL"\|"XXL"` (فقط برای برگرداندن موجودی همون سایز) |
| **Response** | `200 OK` — `{ "id","name","seller_name","description","price","image_url","available_sizes":["M","L","XL","XXL"] }` |
| **Authentication** | Bearer |
| **Error Codes** | `404` · استاندارد |
| **Pagination** | ندارد |

خرید: همان `POST /api/v1/cart/items` با `product_type: "merch"`، `size` هم در body ارسال شود.

```json
{
  "id": "1",
  "name": "Sport T-shirt",
  "seller_name": "Sport Mall 2",
  "description": "Official club home kit jersey.",
  "price": "44,00 €",
  "image_url": "https://cdn.smartklap.com/store/shirt.png",
  "available_sizes": ["M", "L", "XL", "XXL"]
}
```

---

## 8. Video Module

### 8.1 Video Screen — All (feed)

| Field | Value |
|---|---|
| **Screen Name** | Video Screen — All tab |
| **Endpoint** | `/api/v1/videos/feed` |
| **HTTP Method** | GET |
| **Request** | Query: `page?`, `limit?` |
| **Response** | `200 OK` — `{ "items": [{ "id","video_url","thumbnail_url","author":{"name","avatar_url"},"posted_at","tags":[],"likes_count","views_count","is_liked" }], "meta" }` |
| **Authentication** | Bearer |
| **Error Codes** | استاندارد |
| **Pagination** | بله |

**لایک/آنلایک:** `POST /api/v1/videos/{video_id}/like` و `DELETE /api/v1/videos/{video_id}/like`

```json
{
  "items": [
    {
      "id": "v_1",
      "video_url": "https://cdn.smartklap.com/videos/v1.mp4",
      "thumbnail_url": "https://cdn.smartklap.com/videos/v1_thumb.jpg",
      "author": { "name": "Alireza Najari", "avatar_url": "https://cdn.smartklap.com/avatars/u1.png" },
      "posted_at": "2026-07-17T15:00:00Z",
      "tags": ["BestPlayer", "BurgosGame", "BestGameEver"],
      "likes_count": 1236,
      "views_count": 1400,
      "is_liked": false
    }
  ],
  "meta": { "page": 1, "limit": 10, "total": 42, "total_pages": 5 }
}
```

### 8.2 Video Screen — My Videos

| Field | Value |
|---|---|
| **Screen Name** | Video Screen — My videos tab |
| **Endpoint** | `/api/v1/videos/mine` |
| **HTTP Method** | GET |
| **Request** | Query: `page?`, `limit?` |
| **Response** | همان شکل `videos/feed` |
| **Authentication** | Bearer |
| **Error Codes** | استاندارد |
| **Pagination** | بله |

### 8.3 New Post Screen

| Field | Value |
|---|---|
| **Screen Name** | New Post Screen |
| **Endpoint** | `/api/v1/videos` |
| **HTTP Method** | POST |
| **Request** | `multipart/form-data` — فیلدهای: `media` (file, image یا video), `type: "image"\|"video"`, `caption?: "string"` |
| **Response** | `201 Created` — `{ "id","status": "processing"\|"published","video_url" }` |
| **Authentication** | Bearer |
| **Error Codes** | `413` حجم فایل زیاد · `415` فرمت غیرمجاز · استاندارد |
| **Pagination** | ندارد |

```json
{ "id": "v_99", "status": "processing", "video_url": null }
```

---

## 9. Statistics Module

### 9.1 Game Detail Screen

| Field | Value |
|---|---|
| **Screen Name** | Game Detail Screen |
| **Endpoint** | `/api/v1/matches/{match_id}` |
| **HTTP Method** | GET |
| **Request** | Path param `match_id` |
| **Response** | `200 OK` — `{ "home_team","away_team","score","minute","stadium","stats":[{ "label","home","away" }],"timeline":[...],"squads":[{ "title","players":[{ "name","position","photo_url" }] }] }` |
| **Authentication** | Bearer |
| **Error Codes** | `404` · استاندارد |
| **Pagination** | ندارد |

```json
{
  "home_team": { "name": "SP Burgos", "logo_url": "https://cdn.smartklap.com/logos/burgos.png" },
  "away_team": { "name": "FC Barcelona", "logo_url": "https://cdn.smartklap.com/logos/barcelona.png" },
  "score": "3 : 2",
  "minute": "90+6",
  "stadium": "Camp Nou",
  "stats": [
    { "label": "Total shots", "home": 95, "away": 85 },
    { "label": "Possession", "home": 55, "away": 45 }
  ],
  "timeline": [
    { "kind": "marker", "minute": "45+3", "score": "2 - 1" },
    { "kind": "event", "side": "home", "type": "goal", "name": "Dani Alves", "minute": "66", "highlighted": true }
  ],
  "squads": [
    {
      "title": "Forward",
      "players": [
        { "id": "p_1", "name": "Gerard Pique", "position": "Forward", "photo_url": "https://cdn.smartklap.com/players/p1.png" }
      ]
    }
  ]
}
```

### 9.2 Players Screen

| Field | Value |
|---|---|
| **Screen Name** | Players Screen |
| **Endpoint** | `/api/v1/players/{player_id}` |
| **HTTP Method** | GET |
| **Request** | Path param `player_id`; query `match_id?` (برای highlight روی زمین در همان مسابقه) |
| **Response** | `200 OK` — `{ "name","jersey_number","club","age","preferred_foot","nationality","height_cm","weight_kg","weak_foot_percentage","photo_url","radar_stats":[{ "label","value" }],"formation" }` |
| **Authentication** | Bearer |
| **Error Codes** | `404` · استاندارد |
| **Pagination** | ندارد |

> `formation` (مثل `"4-1-2-2"`) فقط اسم فرمیشن است؛ محاسبه‌ی مکان x/y بازیکنان روی زمین سمت فرانت انجام می‌شود (`src/constants/footballFormations.ts`) — لازم نیست بک‌اند کوردینات بفرسته، مگر بخوایم فرمیشن سفارشی (custom x/y) هم ساپورت کنیم که در اون صورت هر بازیکن `x`,`y` هم داره.

```json
{
  "name": "Dani Alves",
  "jersey_number": 8,
  "club": "FC Barcelona",
  "age": 28,
  "preferred_foot": "Right",
  "nationality": "Spain",
  "height_cm": 201,
  "weight_kg": 91,
  "weak_foot_percentage": 64,
  "photo_url": "https://cdn.smartklap.com/players/dani-alves.png",
  "radar_stats": [
    { "label": "Attack", "value": 50 },
    { "label": "Skill", "value": 85 },
    { "label": "Defence", "value": 50 },
    { "label": "Tactic", "value": 50 },
    { "label": "Creativity", "value": 65 }
  ],
  "formation": "4-1-2-2"
}
```

---

## نکات باز برای هماهنگی با بک‌اند (قبل از شروع پیاده‌سازی)

1. **Auth واقعاً OTP است، نه password.** فایل فعلی `src/api/auth.ts` باید به‌روزرسانی شود.
2. هیچ صفحه‌ای فعلاً `id` واقعی در navigation params نمی‌فرستد (مثلاً کلیک روی هر Snack/Product/Quiz به همون Detail Screen مشترک می‌رود بدون id) — این باید هم در فرانت (بعد از API) و هم در طراحی endpoint اصلاح شود.
3. `SoccerClubScreen` (Club Mode Home) فعلاً به هیچ route وصل نیست؛ اگر قرار نیست فعلاً استفاده شود، `home/club` را می‌توان فاز بعد گذاشت.
4. سبد خرید مشترک بین Snacks و Store است (`cart_count` هم‌جا دیده می‌شود) — پیشنهاد یک `cart` واحد با `product_type` به جای دو سبد جدا.
5. Currency toggle (Euro/Point) در Snacks و Store فعلاً روی UI اثر ندارد — باید مشخص شود قیمت Point چطور محاسبه می‌شود.
6. `ArBanner` (روی Home) به مسیر `/(tabs)/AR` push می‌کند ولی این route اصلاً در پروژه ساخته نشده (فولدر `src/app/AR/` خالی است) — این یک صفحه/فیچر جداست (AR) که هنوز نه UI و نه API برایش تعریف نشده؛ اگر در اسکوپ فعلی نیست بهتره از این سند و بک‌لاگ فعلی حذف بشه یا جدا مشخص شود.
