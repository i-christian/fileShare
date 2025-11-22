# API usage flow

### 🔐 Authentication Flow (Testing with cURL)

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

#### Load testing using `hey` package
- We can use a tool like hey to generate some requests to our application and see its performance under load. For example, we can send a batch of requests to the POST /api/v1/auth/login endpoint (which is slow and costly because it checks a bcrypt-hashed password) as follows:

1. Install [hey](https://github.com/rakyll/hey?tab=readme-ov-file) on linux you can use:
  ```
    sudo apt update && sudo apt install hey
  ```
2. Run with the api running with rate limiter disabled using:
```
  go run ./cmd/api -limiter-enabled=false
```
3. Test the login endpoint as this example:
```
  BODY='{"email": "alice@example.com", "password": "supersecret123"}'
  
  hey -d "$BODY" -m "POST" http://localhost:8080/api/v1/auth/login
```

**Response:**
```
  Summary:
  Total:        13.3192 secs
  Slowest:      3.8290 secs
  Fastest:      2.2823 secs
  Average:      3.2493 secs
  Requests/sec: 15.0159

  Total data:   86600 bytes
  Size/request: 433 bytes

Response time histogram:
  2.282 [1]     |■
  2.437 [0]     |
  2.592 [1]     |■
  2.746 [7]     |■■■■■■
  2.901 [13]    |■■■■■■■■■■■■
  3.056 [30]    |■■■■■■■■■■■■■■■■■■■■■■■■■■■
  3.210 [31]    |■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  3.365 [45]    |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  3.520 [33]    |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  3.674 [30]    |■■■■■■■■■■■■■■■■■■■■■■■■■■■
  3.829 [9]     |■■■■■■■■


Latency distribution:
  10% in 2.8810 secs
  25% in 3.0533 secs
  50% in 3.2646 secs
  75% in 3.4607 secs
  90% in 3.6082 secs
  95% in 3.6706 secs
  99% in 3.7771 secs

Details (average, fastest, slowest):
  DNS+dialup:   0.0018 secs, 2.2823 secs, 3.8290 secs
  DNS-lookup:   0.0012 secs, 0.0000 secs, 0.0143 secs
  req write:    0.0007 secs, 0.0001 secs, 0.0127 secs
  resp wait:    3.2460 secs, 2.2616 secs, 3.8278 secs
  resp read:    0.0006 secs, 0.0001 secs, 0.0065 secs

Status code distribution:
  [200] 200 responses
```

-----

# 📂 File Management

Ensure you have a valid `$ACCESS_TOKEN` exported from the authentication steps above before proceeding.

-----

## 7️⃣ Upload a File
**NOTE:** image upload generate thumbnails automatically.
First, create a dummy file to test with:

```bash
echo "Hello, this is a test document for fileShare!" > test_doc.txt
```

Now, upload it using `multipart/form-data`:

```bash
curl -X POST http://localhost:8080/api/v1/files/upload \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -F "file=@./test_doc.txt"
```

**Response:**

```json
{
        "file": {
                "file_id": "019aabc7-9fdb-7003-b850-a02760b78401",
                "filename": "test_doc.txt",
                "mime_type": "text/plain; charset=utf-8",
                "size_bytes": 46,
                "created_at": "2025-11-22T15:36:17.882173+02:00",
                "visibility": "private",
                "checksum": "008aa47eb4515f3974d9558b0bafe981eaaa5852950ead221fa9ffef09fe959a",
                "version": 0
        },
        "message": "File uploaded successfully"
}
```

👉 **Save the File ID** for the next steps:

```bash
export FILE_ID="019aabc7-9fdb-7003-b850-a02760b78401"
```

-----

#### Image files and thumbnails
```bash
curl -X POST http://localhost:8080/api/v1/files/upload \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -F "file=@./image.jpg"
```


## 8️⃣ List "My Files" (Private & Public)

This endpoint lists only the files uploaded by the authenticated user. It supports pagination.

```bash
curl -X GET "http://localhost:8080/api/v1/files/me?page=1&page_size=1" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

**Response:**

```json
{
  "files": [
    {
      "file_id": "019aa7de-97b0-7d6b-af94-5a0e203ece41",
      "filename": "test_doc.txt",
      "mime_type": "application/octet-stream",
      "size_bytes": 46,
      "visibility": "private",
      "created_at": "2025-11-21T21:22:54.256243+02:00",
      "tags": []
    }
  ],
  "metadata": {
    "current_page": 1,
    "page_size": 5,
    "first_page": 1,
    "last_page": 1,
    "total_records": 1
  }
}
```

-----

## 9️⃣ List Public Files (The Feed)

This endpoint is accessible to **everyone** (authenticated or anonymous). It only shows files marked as `public`.

```bash
curl -X GET "http://localhost:8080/api/v1/files?page=1&page_size=5"
```

**Response:**
```
  {
  "files": [
    {
      "owner_id": "019a5ac0-e9a0-7645-91d7-4cab346fef34",
      "last_name": "Wonderland",
      "first_name": "alice",
      "file_id": "019aa7de-97b0-7d6b-af94-5a0e203ece41",
      "filename": "test_doc.txt",
      "mime_type": "application/octet-stream",
      "size_bytes": 46,
      "thumbnail_key": {
        "String": "",
        "Valid": false
      },
      "checksum": "008aa47eb4515f3974d9558b0bafe981eaaa5852950ead221fa9ffef09fe959a",
      "tags": [],
      "version": 0
    }
  ],
  "metadata": {
    "current_page": 1,
    "page_size": 5,
    "first_page": 1,
    "last_page": 1,
    "total_records": 1
  }
}
```
-----

## 10 Get File Metadata

Retrieves details about a specific file.
  * You must be the owner and authenticated

```bash
curl -X GET http://localhost:8080/api/v1/files/$FILE_ID \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

**Response:**

```json
{
  "file": {
    "file_id": "019aa7de-97b0-7d6b-af94-5a0e203ece41",
    "owner_id": "019a5ac0-e9a0-7645-91d7-4cab346fef34",
    "filename": "test_doc.txt",
    "mime_type": "application/octet-stream",
    "storage_key": "users/019a5ac0-e9a0-7645-91d7-4cab346fef34/395eccb4-191d-4eba-a480-6de7f1ff1763.txt",
    "size_bytes": 46,
    "visibility": "public",
    "thumbnail_key": {
      "String": "",
      "Valid": false
    },
    "checksum": "008aa47eb4515f3974d9558b0bafe981eaaa5852950ead221fa9ffef09fe959a",
    "tags": [],
    "version": 1
  }
}
```

-----
## 11 Change file visibility
This allows the file owner to change the file visibility status.
* Visibility status is either `public` or `private`
```bash
curl -X PUT http://localhost:8080/api/v1/files/$FILE_ID/visible \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{
        "version": 1,
        "visibility": "public"
  }'
```

**Response:**
```
{
    "message": "file visibility has been updated to public"
}
```

Try checking the file metadata again, visibility should be public now and the version updated too.

-----
## 12 Change filename
This allows the file owner to change the file visibility status.
```bash
curl -X PUT http://localhost:8080/api/v1/files/$FILE_ID/edit \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{
        "version": 2,
        "filename": "renamed_file.txt"
  }'
```

**Response:**
```
{
    "message": "file visibility has been updated to public"
}
```

Try checking the file metadata again, visibility should be public now and the version updated too.

-----
## 13 Download a File

This streams the binary content of the file. We use the `--output` flag to save it locally instead of printing binary data to the console.
***NOTE:*** Authorization header is optional here, its however used to allow file owners to download their own files even if the files are private.
```bash
curl -X GET http://localhost:8080/api/v1/files/$FILE_ID/download \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  --output downloaded_test.txt
```

OR

```bash
  curl -X GET http://localhost:8080/api/v1/files/$FILE_ID/download \
  --output downloaded_test.txt
```

**Verify the content:**

```bash
cat downloaded_test.txt
# Output: Hello, this is a test document for fileShare!
```

-----

## 14 Delete a File

Permanently marks a file as deleted. Only the owner can do this.
```bash
curl -X PUT http://localhost:8080/api/v1/files/$FILE_ID \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "version": 1
  }'

```

**Response:**

```json
{
    "message": "file deleted successfully"
}
```

If you try to `GET` the metadata of this file again, you should receive a **404 Not Found**.
```bash
curl -v http://localhost:8080/api/v1/files/$FILE_ID \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

**Response:**
```json
{
  "error": "the record does not exist"
}
```
