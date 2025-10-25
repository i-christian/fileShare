# 🔐 Authentication Flow (Testing with cURL)

Below is a full working example showing how to test user signup, login, and token refresh using `curl`.

> 💡 Replace `localhost:8080` with your actual API base URL if different.
> 💡 Tokens are stored in shell environment variables (they disappear when you close your terminal).

---

## 1️⃣ Register a new user

```bash
curl -X POST http://localhost:8080/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "first_name": "Alice",
    "last_name": "Wonderland",
    "password": "supersecret123"
  }'
```

**Response:**

```json
{
  "user_id":"019a170b-75e0-71a3-8704-7dd53dd2ae5d",
  "last_name":"Wonderland",
  "first_name":"Alice",
  "email":"alice@example.com",
  "is_verified":false,
  "role":"user",
  "updated_at":"2025-10-24T18:26:58.399625+02:00",
  "last_login":{"Time":"2025-10-24T18:26:58.399625+02:00","Valid":true}
}
```

---

## 2️⃣ Login and receive access + refresh tokens

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "password": "supersecret123"
  }'
```

**Response:**

```json
{
  "access_token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImFsaWNlQGV4YW1wbGUuY29tIiwiZXhwIjoxNzYxMjI1NDYwLCJmaXJzdF9uYW1lIjoiQWxpY2UiLCJpYXQiOjE3NjEyMjQ1NjAsImxhc3RfbmFtZSI6IldvbmRlcmxhbmQiLCJzdWIiOiIwMTlhMTEyOC00ZjE4LTc1YmQtODczMS1iZDZiOGMwMmFlODYifQ.7dxSsj9D7-2LMqSo-GEm2r_iISUsgqGLtMu4RknImUc",
  "refresh_token":"ea3b4681-2df7-41f4-a3c1-0ff66653fa9a"
}
```

👉 Save these tokens as temporary environment variables (they’ll disappear when the shell closes):

```bash
export ACCESS_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImFsaWNlQGV4YW1wbGUuY29tIiwiZXhwIjoxNzYxMjI1NDYwLCJmaXJzdF9uYW1lIjoiQWxpY2UiLCJpYXQiOjE3NjEyMjQ1NjAsImxhc3RfbmFtZSI6IldvbmRlcmxhbmQiLCJzdWIiOiIwMTlhMTEyOC00ZjE4LTc1YmQtODczMS1iZDZiOGMwMmFlODYifQ.7dxSsj9D7-2LMqSo-GEm2r_iISUsgqGLtMu4RknImUc"
export REFRESH_TOKEN="ea3b4681-2df7-41f4-a3c1-0ff66653fa9a"
```

---

## 3️⃣ Access a protected endpoint using the access token

```bash
curl -X GET http://localhost:8080/api/v1/user/me \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

**Response (if token valid):**

```json
{
  "user_id":"019a1b18-609c-723b-b20e-903b83d73941",
  "email":"alice@example.com",
  "first_name":"Alice",
  "last_name":"Wonderland",
  "is_verified":false,
  "role":"user"
}
```

**Response (if token expired):**

```json
{
  "error": "token has expired"
}
```

---

## 4️⃣ Refresh your access token

When your access token expires, request a new one using your refresh token:
```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d "{\"refresh_token\": \"$REFRESH_TOKEN\"}"
```

**Response:**

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

👉 Save the new access token:

```bash
export ACCESS_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

---

## 5️⃣ Retry the protected endpoint with the new token

```bash
curl -X GET http://localhost:8080/api/v1/user/me \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

**Response:**

```json
{
  "user_id":"019a1b18-609c-723b-b20e-903b83d73941",
  "email":"alice@example.com",
  "first_name":"Alice",
  "last_name":"Wonderland",
  "is_verified":false,
  "role":"user"
}
```

---
