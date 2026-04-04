# User Session Management Specification

## 1. Overview

This document specifies how user sessions are managed in Free Orgx. Sessions track authenticated users across HTTP requests using cookie-based authentication.

## 2. Current Implementation

### 2.1 Session Cookie

| Property | Value |
|----------|-------|
| Cookie Name | `session_id` |
| Cookie Type | HTTP-only |
| Path | `/` |
| Token Format | 32-byte random string, base64-encoded |

### 2.2 Session Lifecycle

1. **Login** (`POST /submit`): User credentials validated, session token generated and stored in database
2. **Authentication**: Each protected request validates token against database
3. **Logout** (`GET /logout`): Token cleared from database, cookie expired
4. **Expiration**: 60 minutes (SQLite), 1 minute (RAM mock)

### 2.3 Token Storage

Session tokens are stored in the `users` table:

```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    token TEXT,           -- session token
    role TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 2.4 Validation Flow

```go
// 1. Extract cookie
c, err := r.Cookie("session_id")

// 2. Validate token against database
user, err := db.IsValid(ctx, c.Value)

// 3. Use user info for authorization
if user.Role == ports.RoleAdmin { /* allow admin action */ }
```

## 3. Security Measures

1. **Token Generation**: `crypto/rand` generates 32 random bytes
2. **HttpOnly Cookie**: Prevents JavaScript access (XSS protection)
3. **Bcrypt Password Hashing**: With pepper suffix
4. **Role Checks**: Admin routes verify user role before processing

## 4. Session vs Token

- **Session**: HTTP concept (cookie with expiration)
- **Token**: Database value that maps to user

The cookie contains the token. The session expiration is handled by browser (cookie expiry) and server (token validation with timestamp).

## 5. Configuration

| Environment | Token Expiry |
|-------------|---------------|
| SQLite | 60 minutes |
| RAM Mock | 1 minute |

## 6. Related Code

- `internal/ports/db.go`: `SessionCookie`, `User` structs
- `internal/handlers/handlers.go`: Session cookie handling
- `internal/adapters/database/sqLite.go`: Token storage/validation
- `internal/adapters/mocks/ram.go`: In-memory session validation
