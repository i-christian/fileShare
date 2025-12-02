# 📁 fileShare API — Modern File Management Backend

fileShare API is a **backend service** built with **Go**, providing secure, scalable, and performant file management.  
It exposes a **RESTful JSON API** that handles authentication, file uploads, downloads, and background processing — designed for integration with any frontend (web, mobile, CLI, etc.).

---

## ✨ Features

- 🔐 **JWT Authentication** – Secure stateless authentication with refresh tokens.
- 🚦 **Rate Limiting** – Protects the API from abuse using per-user/IP limits.
- 🗃️ **PostgreSQL Storage** – Reliable relational database for metadata.
- ⚙️ **Redis Integration** – Caching and background job queue.
- 🧵 **Concurrent Background Workers** – For thumbnails, virus scans, or cleanup tasks.
- 🧰 **Docker-Ready** – Containerized with Docker Compose for easy setup.

---

## 🏗️ Architecture Overview

```mermaid
flowchart LR
    subgraph Clients
        A[Web / Mobile / CLI]
    end

    subgraph API Server
        B[Chi Router + JSON Handlers]
        C[JWT Middleware]
        D[Rate Limiter]
        E[File Handlers]
        F[Background Worker]
    end

    subgraph Services
        G[(PostgreSQL)]
        H[(Redis)]
        I[(Object Storage: Local/S3)]
    end

    A -->|HTTP JSON| B
    B --> C
    B --> D
    C --> E
    E --> G
    E --> H
    E --> I
    F --> H
    F --> I
````

---

## ⚙️ Tech Stack

| Component            | Technology                                            | Description                          |
| -------------------- | ----------------------------------------------------- | ------------------------------------ |
| **Language**         | Go (Golang)                                           | High-performance, type-safe backend. |
| **Router**           | [Chi](https://github.com/go-chi/chi)                  | Lightweight idiomatic HTTP router.   |
| **Auth**             | [JWT (golang-jwt)](https://github.com/golang-jwt/jwt) | Stateless user authentication.       |
| **Database**         | PostgreSQL                                            | Primary relational data store.       |
| **Task Queue**       | [Asynq](https://github.com/hibiken/asynq.git)         | A task queue(redis backed).          |
| **Containerization** | Docker & Docker Compose                               | Consistent dev/prod environments.    |

---

## 🧩 Core API Endpoints

| Method   | Endpoint                       | Description                       | Auth       |
| -------- | ------------------------------ | --------------------------------- | ---------- |
| `GET`    | `/api/v1/healthcheck`          | Check the application status      | ❌         |
| `POST`   | `/api/v1/auth/signup`          | Register a new user               | ❌         |
| `POST`   | `/api/v1/auth/login`           | Login and get JWT tokens          | ❌         |
| `POST`   | `/api/v1/auth/password/recover`| Send password reset email         | ❌         |
| `PUT`    | `/api/v1/auth/password/reset`  | Set new password                  | ❌         |
| `POST`   | `/api/v1/auth/refresh`         | Refresh JWT token                 | ✅         |
| `PUT`    | `/api/v1/user/activated`       | Verify email                      | ✅         |
| `GET`    | `/api/v1/user/me`              | Get current user profile          | ✅         |
| `POST`   | `/api/v1/user/api-keys`        | Create an API Key                 | ✅         |
| `POST`   | `/api/v1/files/upload`         | Upload new file (supports chunks) | ✅         |
| `GET`    | `/api/v1/files`                | List public files                 | ❌         |
| `GET`    | `/api/v1/files/me`             | List user files                   | ✅         |
| `GET`    | `/api/v1/files/{id}`           | Get file metadata                 | ✅         |
| `GET`    | `/api/v1/files/{id}/download`  | Download file                     | ❌         |
| `DELETE` | `/api/v1/files/{id}`           | Delete file                       | ✅         |
| `PUT`    | `/api/v1/files/{id}/visible`   | Change file visibility            | ✅         |
| `PUT`    | `/api/v1/files/{id}/edit`      | Change filename                   | ✅         |

---

## 🧰 Development Setup
For instructions on how to get started with this application, please refer to the [Development Documentation](/development.md).

This documentation provides instructions on how to set up your environment and develop the application.

---
## Endpoints Testing 
For basic curl commands to test the api endpoints refer to [API USAGE](/usage.md).

This documentation provide simple curl commands to test the api without need for postman. However you can also use Postman if you prefer than.

---

## 🚀 Roadmap
* [ ] Implement file versioning
* [ ] Add virus scanning worker
* [ ] Integrate Digital Ocean spaces
