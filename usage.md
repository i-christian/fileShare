# 🔐 Authentication Flow (Testing with cURL)

Below is a full working example showing how to test user signup, login, and token refresh using `curl`.

> 💡 Replace `localhost:8080` with your actual API base URL if different.
> 💡 Tokens are stored in shell environment variables (they disappear when you close your terminal).

---

## 1️⃣ Register a new user

```bash
curl -v http://localhost:8080/api/v1/auth/signup \
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
        "user": {
                "last_login": {
                        "Time": "2025-11-02T14:34:12.918644+02:00",
                        "Valid": true
                },
                "last_name": "Wonderland",
                "first_name": "Alice",
                "email": "alice@example.com",
                "role": "user",
                "user_id": "019a448f-9938-764b-a1c8-a22b8ce3bd45",
                "is_verified": false
        }
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
        "tokens": {
                "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImFsaWNlQGV4YW1wbGUuY29tIiwiZXhwIjoxNzYyMDg3ODEzLCJmaXJzdF9uYW1lIjoiQWxpY2UiLCJpYXQiOjE3NjIwODY5MTMsImxhc3RfbmFtZSI6IldvbmRlcmxhbmQiLCJzdWIiOiIwMTlhNDQ4Zi05OTM4LTc2NGItYTFjOC1hMjJiOGNlM2JkNDUifQ.fFB1xqhvfIjQb2j0tOBfbGgSrY2lVJ8QPeJDGBfOnrU",
                "refresh_token": "7e7ec54e-d4f5-437f-9753-c58457cca384"
        }
}
```

👉 Save these tokens as temporary environment variables (they’ll disappear when the shell closes):

```bash
export ACCESS_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImFsaWNlQGV4YW1wbGUuY29tIiwiZXhwIjoxNzYyMDg3ODEzLCJmaXJzdF9uYW1lIjoiQWxpY2UiLCJpYXQiOjE3NjIwODY5MTMsImxhc3RfbmFtZSI6IldvbmRlcmxhbmQiLCJzdWIiOiIwMTlhNDQ4Zi05OTM4LTc2NGItYTFjOC1hMjJiOGNlM2JkNDUifQ.fFB1xqhvfIjQb2j0tOBfbGgSrY2lVJ8QPeJDGBfOnrU"
export REFRESH_TOKEN="7e7ec54e-d4f5-437f-9753-c58457cca384"
```

---

### Verify user email
```bash
curl -X PUT http://localhost:8080/api/v1/user/activated \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN"\
  -d '{
       "token": "WKTJKOIUVTUXOBZ4EFO66FB55V"
  }'
```

**Response (if token valid):**
```
  {
        "user": {
                "user_id": "019a4b3e-18d6-74a3-991d-0c42f2d344ec",
                "email": "alice@example.com",
                "first_name": "Alice",
                "last_name": "Wonderland",
                "is_verified": true,
                "role": "user"
        }
}
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
        "user": {
                "user_id": "019a448f-9938-764b-a1c8-a22b8ce3bd45",
                "email": "alice@example.com",
                "first_name": "Alice",
                "last_name": "Wonderland",
                "is_verified": false,
                "role": "user"
        }
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
  "user_id":"019a1b18-609c-723b-b20e-903b83d73941",
  "email":"alice@example.com",
  "first_name":"Alice",
  "last_name":"Wonderland",
  "is_verified":false,
  "role":"user"
}
```

---

### 6️⃣ Create a New API Key 

#### with expiration field
```bash
curl -X POST http://localhost:8080/api/v1/user/api-keys \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{
    "key_name": "CLI Integration Key",
    "expires_at": "2025-12-31T23:59:59Z",
    "scope": ["read", "write"]
  }'
```

**Response:**

```json
{
  "apiKey": "fi068eqB7p5q9H_HOmSckSiUQBNlbUPyLO7PNAEKTN560ga"
}
```

#### without expiration field
```bash
curl -X POST http://localhost:8080/api/v1/user/api-keys \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{
    "key_name": "CLI Integration Key",
    "scope": ["read", "write"]
  }'
```


**Response:**

```json
{
  "apiKey": "fi068eXkGfaHQr_gfexbYIauupjsYf3XA3enXponoO2t8F0"
}
```

You can store it and authenticate future requests using:

```bash
export API_KEY="fi068eXkGfaHQr_gfexbYIauupjsYf3XA3enXponoO2t8F0"
```

Then access any API endpoint using the key:

```bash
curl -X GET http://localhost:8080/api/v1/user/me \
  -H "Authorization: ApiKey $API_KEY"
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
