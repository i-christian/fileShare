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
  "user_id":"019a1128-4f18-75bd-8731-bd6b8c02ae86",
  "last_name":"Wonderland",
  "first_name":"Alice",
  "email":"alice@example.com",
  "is_verified":false,
  "role":"user",
  "password_hash":"$2a$10$m8B.xBlDlrh/YgSJ6pCqMeRys9V30XAZGZTLEbTl.okZiLE50M2V2",
  "created_at":"2025-10-23T15:00:45.718372+02:00",
  "updated_at":"2025-10-23T15:00:45.718372+02:00",
  "last_login":{"Time":"2025-10-23T15:00:45.718372+02:00","Valid":true}
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
  "user_id": "1d8b8f9e-1f4b-4e8c-8a60-7cc6a35f29f2",
  "email": "alice@example.com",
  "first_name": "Alice",
  "last_name": "Wonderland"
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
  "user_id": "1d8b8f9e-1f4b-4e8c-8a60-7cc6a35f29f2",
  "email": "alice@example.com",
  "first_name": "Alice",
  "last_name": "Wonderland"
}
```

---
