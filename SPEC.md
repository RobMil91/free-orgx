# Free Orgx - Project Specification Document

## Table of Contents
1. [Project Overview](#1-project-overview)
2. [Architecture](#2-architecture)
3. [Features/API](#3-featuresapi)
4. [Data Models](#4-data-models)
5. [Database](#5-database)
6. [Configuration](#6-configuration)
7. [Security](#7-security)
8. [Error Handling](#8-error-handling)

---

## 1. Project Overview

**Free Orgx** is a minimalistic organization tool built with Go that provides user management and project creation capabilities. It uses server-side HTML rendering with HTMX for dynamic interactions.

### Key Characteristics
- **Server-Side Rendering**: HTML templates rendered on the server
- **Database**: SQLite for persistent storage (with RAM mock for testing)
- **Session-Based Authentication**: Cookie-based session tokens
- **Role-Based Access Control**: Admin and User roles

### Technology Stack
| Component | Technology |
|-----------|------------|
| Language | Go 1.24.0 |
| Database | SQLite (github.com/mattn/go-sqlite3) |
| Password Hashing | bcrypt (golang.org/x/crypto) |
| Templating | Go standard library `html/template` |
| Frontend | HTMX (htmx.min.js) |
| Logging | Go standard library `log/slog` |

---

## 2. Architecture

### 2.1 Directory Structure

```
/home/ubu/projects/free-orgx/
├── main.go                      # Application entry point
├── config/
│   └── conf.go                  # Configuration struct
├── internal/
│   ├── models/
│   │   ├── user.go              # User model (empty package)
│   │   └── project.go           # Project model struct
│   ├── ports/
│   │   └── db.go                # Repository interfaces and error types
│   ├── handlers/
│   │   ├── handlers.go          # HTTP handler functions
│   │   └── handlers_test.go     # Handler tests
│   ├── adapters/
│   │   ├── database/
│   │   │   └── sqLite.go        # SQLite adapter implementation
│   │   ├── mocks/
│   │   │   └── ram.go           # In-memory RAM database mock
│   │   └── htmlx/
│   │       ├── login.html       # Login template
│   │       ├── project.html      # Projects page template
│   │       └── admin_users.html  # Admin user management template
│   └── setup/
│       └── setup.go             # Dependency injection/setup
├── static/
│   ├── index.html               # Home page
│   └── htmx.min.js              # HTMX library
└── orgxdb.db                    # SQLite database file (runtime)
```

### 2.2 Main Components

#### Configuration (`config/conf.go`)
```go
type Config struct {
    Port  string  // Server port (default: "8080")
    RAMDB bool    // Use RAM mock instead of SQLite
}
```

#### Ports - Interfaces (`internal/ports/db.go`)
Defines repository interfaces that adapters must implement:
- **`UserRepo`**: User authentication and management
- **`ProjectRepo`**: Project CRUD operations

#### Adapters
| Adapter | File | Description |
|---------|------|-------------|
| SQLite | `internal/adapters/database/sqLite.go` | Production database adapter |
| RAM | `internal/adapters/mocks/ram.go` | In-memory mock for testing |

#### Models (`internal/models/`)
- **`User`**: (defined in ports/db.go) - User entity with authentication data
- **`Project`**: Project entity with ID, Name, Owner, Created timestamp

#### Setup (`internal/setup/setup.go`)
Dependency injection container that initializes appropriate adapters based on configuration.

#### Handlers (`internal/handlers/handlers.go`)
HTTP handler functions that process incoming requests:
- **`LoginHandler`**: Display login form
- **`LogoutHandler`**: Logout and clear session
- **`ProjectHandler`**: Display user's projects
- **`LoginSubmit`**: Process login credentials
- **`CreateProjectHandler`**: Create a new project
- **`AdminUsersHandler`**: List all users (admin only)
- **`AdminCreateUserHandler`**: Create new user (admin only)

### 2.3 Component Interaction

```
HTTP Request
    │
    ▼
main.go (Application Entry Point)
    │
    ▼
internal/handlers (HTTP Handler Functions)
    │
    ├── Requires session cookie
    │
    ▼
ports.UserRepo / ports.ProjectRepo (Interfaces)
    │
    ▼
Adapters (SQLiteAdapter or RAM)
    │
    ▼
Database / In-Memory Storage
```

---

## 3. Features/API

### 3.1 HTTP Endpoints

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/` | FileServer | Serve static files (index.html) |
| GET | `/login` | `handlers.LoginHandler` | Display login form |
| POST | `/submit` | `handlers.LoginSubmit` | Process login credentials |
| GET | `/project` | `handlers.ProjectHandler` | Display user's projects |
| POST | `/project/create` | `handlers.CreateProjectHandler` | Create a new project |
| GET | `/logout` | `handlers.LogoutHandler` | Logout and clear session |
| GET | `/admin/users` | `handlers.AdminUsersHandler` | Admin: List all users |
| POST | `/admin/user/create` | `handlers.AdminCreateUserHandler` | Admin: Create new user |

### 3.2 Endpoint Details

#### GET `/login`
- **Purpose**: Display login form
- **Auth Required**: No
- **Response**: HTML login form (login.html template)

#### POST `/submit`
- **Purpose**: Process login submission
- **Auth Required**: No
- **Form Parameters**:
  - `user`: Username
  - `password`: Password
- **Response**: 
  - Success: Sets `session_id` cookie, returns welcome message for first admin
  - Failure: Error message

#### GET `/project`
- **Purpose**: Display user's projects page
- **Auth Required**: Yes (valid session cookie)
- **Response**: HTML with project list (project.html template)

#### POST `/project/create`
- **Purpose**: Create a new project
- **Auth Required**: Yes
- **Form Parameters**:
  - `name`: Project name (required)
- **Response**: Success message with project ID

#### GET `/logout`
- **Purpose**: Logout current user
- **Auth Required**: Yes
- **Response**: Redirects to home page, clears session cookie

#### GET `/admin/users`
- **Purpose**: Display user management page
- **Auth Required**: Yes
- **Authorization**: Admin role only
- **Response**: HTML with user list (admin_users.html template)

#### POST `/admin/user/create`
- **Purpose**: Create a new user
- **Auth Required**: Yes
- **Authorization**: Admin role only
- **Form Parameters**:
  - `username`: New user's username (required)
  - `password`: New user's password (required)
- **Response**: Success or error message

### 3.3 User Management Features

1. **First-User Admin**: First user to register becomes admin automatically
2. **Login/Logout**: Session-based authentication
3. **User Creation**: Admins can create new users with "user" role
4. **User Listing**: Admins can view all registered users
5. **Password Change**: Users can change their password

### 3.4 Project Management Features

1. **Create Project**: Authenticated users can create projects
2. **List Projects**: Users can view their own projects
3. **Delete Project**: Available via adapter (not exposed in HTTP)

### 3.5 Session Handling

- **Cookie Name**: `session_id`
- **Cookie Properties**:
  - `HttpOnly: true`
  - `Path: /`
- **Token Format**: 32-character base64-encoded random string
- **Token Expiration**: 60 minutes (SQLite) / 1 minute (RAM mock)

---

## 4. Data Models

### 4.1 User Structure

Defined in `internal/ports/db.go`:

```go
type User struct {
    ID       int           // Auto-increment primary key
    Name     string        // Username (unique)
    Role     string        // "admin" or "user"
    Password string        // Password hash (adapter-specific)
    Cookie   SessionCookie // Current session token
}

type SessionCookie struct {
    Value      string    // Session token value
    CreateTime time.Time // Token creation timestamp
}

type LoginResult struct {
    Cookie    *SessionCookie // Generated session cookie
    User      *User          // User information
    IsNewUser bool           // True if this is the first admin
}
```

### 4.2 Roles

| Role | Description | Capabilities |
|------|-------------|--------------|
| `admin` | Administrator | View all users, create new users, access admin panel |
| `user` | Regular user | Create/manage own projects |

Defined as constants:
```go
const (
    RoleAdmin = "admin"
    RoleUser  = "user"
)
```

### 4.3 Project Structure

Defined in `internal/models/project.go`:

```go
type Project struct {
    ID      string // 16-character base64-encoded unique identifier
    Name    string // Project name
    Owner   string // Owner's username
    Created string // Creation timestamp (RFC3339 format)
}
```

### 4.4 Session Cookies

The session cookie contains only the token value. User data is validated against the database using this token.

| Property | Value |
|----------|-------|
| Name | `session_id` |
| Type | HTTP-only cookie |
| Path | `/` |
| Expiry | 60 minutes (SQLite), 1 minute (RAM mock) |

---

## 5. Database

### 5.1 SQLite Schema

Tables are created automatically in `SQLiteAdapter.CreateTables()`:

#### Users Table
```sql
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    token TEXT,
    role TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

#### Projects Table
```sql
CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    owner TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 5.2 Database File Location

| Environment | File Location |
|-------------|---------------|
| Production | `orgxdb.db` (in working directory) |
| Test | `test_<name>.db` |

### 5.3 RAM Mock Database

The in-memory mock (`internal/adapters/mocks/ram.go`) implements both `UserRepo` and `ProjectRepo` interfaces:

```go
type RAM struct {
    Users    map[string]ports.User           // Username -> User mapping
    Projects map[string][]models.Project    // Owner -> Projects mapping
}
```

**Default Test User** (when RAMDB is enabled):
- Username: `testadmin`
- Password: `testpass`
- Role: `admin`

---

## 6. Configuration

### 6.1 Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `PASSWORD_PEPPER` | **Yes** | Secret pepper string for password hashing |
| `LOG_LEVEL` | No | Logging level (`debug` for debug, anything else for info) |

### 6.2 CLI Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-port` | string | `"8080"` | HTTP server port |
| `-ram` | boolean | `false` | Use RAM mock database instead of SQLite |

### 6.3 Usage Examples

```bash
# Run with default port (8080)
PASSWORD_PEPPER=mysecret go run main.go

# Run on custom port
PASSWORD_PEPPER=mysecret go run main.go -port 3000

# Run with RAM database (testing)
PASSWORD_PEPPER=mysecret go run main.go -ram

# Run with debug logging
LOG_LEVEL=debug PASSWORD_PEPPER=mysecret go run main.go
```

---

## 7. Security

### 7.1 Password Hashing

Passwords are hashed using **bcrypt** with a pepper suffix:

```go
// Combined password = user_password + pepper
combined := password + pepper
hash, err := bcrypt.GenerateFromPassword([]byte(combined), bcrypt.DefaultCost)
```

**Security Features**:
- Uses bcrypt with `DefaultCost` (10 rounds)
- Pepper is appended to password before hashing
- Pepper must be set via `PASSWORD_PEPPER` environment variable

### 7.2 Session Management

1. **Token Generation**: Cryptographically secure random 32-byte string, base64-encoded
2. **Token Storage**: Stored in database (`token` column)
3. **Token Expiration**: 60 minutes (SQLite) / 1 minute (RAM)
4. **Cookie Flags**: HTTP-only and Path=/ set for security

### 7.3 Admin-Only Routes

Protected endpoints check user role before processing:

```go
if user.Role != ports.RoleAdmin {
    w.Write([]byte("Access denied: admin only"))
    return
}
```

**Admin-Only Endpoints**:
- `GET /admin/users` - View all users
- `POST /admin/user/create` - Create new user

### 7.4 First Admin Registration

When no users exist in the database, the first user to register becomes admin automatically:

```go
if userCount == 0 {
    return s.registerFirstAdmin(ctx, name, password, pepper)
}
```

---

## 8. Error Handling

### 8.1 Custom Error Types

Defined in `internal/ports/db.go`:

| Error | Value | Usage |
|-------|-------|-------|
| `UserNotFound` | `"[no such user]"` | User does not exist |
| `AdminExists` | `"[admin already exists]"` | Attempted to create second admin |
| `InvalidRole` | `"[invalid role]"` | Invalid role specified |
| `DatabaseError` | `"database adapter failed to execute action"` | General database failure |

### 8.2 Error Handling in HTTP Handlers

Handlers return appropriate HTTP status codes:

| Error Condition | HTTP Status |
|-----------------|-------------|
| No session cookie | 401 Unauthorized |
| Invalid/expired session | 403 Forbidden |
| Database error | 500 Internal Server Error |
| Missing required field | 400 Bad Request (via response text) |

### 8.3 Logging

- Uses structured logging via `log/slog`
- Log levels: Debug, Info, Error
- Debug logging enabled via `LOG_LEVEL=debug` environment variable

---

## Testing

### Test Coverage
- **Total Coverage**: 87.5%
- **Main Handlers**: 100%
- **SQLite Adapter**: 87.7%
- **RAM Mock**: 96.3%
- **Setup**: 92.3%

### Running Tests
```bash
# Run all tests
go test ./...

# Run with coverage
go test ./... -cover

# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Test Files
| File | Package | Description |
|------|---------|-------------|
| `internal/handlers/handlers_test.go` | `handlers_test` | Handler tests using real database and mocks |
| `internal/adapters/database/sqLite_test.go` | `database_test` | SQLite adapter tests |
| `internal/adapters/mocks/ram_test.go` | `mocks_test` | RAM mock tests |
| `internal/setup/setup_test.go` | `setup_test` | Setup function tests |

---

## Summary

This specification covers all aspects of the Free Orgx application. The project follows clean architecture principles with clear separation between:

- **Ports**: Interface definitions (`internal/ports/db.go`)
- **Adapters**: Implementations (SQLite, RAM mock)
- **Models**: Domain entities
- **Handlers**: HTTP request processing (`internal/handlers/handlers.go`)

The application provides a complete user authentication system with role-based access control, project management capabilities, and secure password handling using bcrypt with pepper.
