# User Roles Feature Specification

## Overview

The application supports two user roles: `admin` and `user`. Only one admin can exist at a time, ensuring a single point of administration.

## Roles

| Role   | Description                              |
|--------|------------------------------------------|
| admin  | Administrator with elevated privileges    |
| user   | Regular user with standard access         |

## Constraints

- Only **one admin** can exist in the system at any time
- Attempting to create a second admin returns `AdminExists` error
- Users can be created without restriction

## Database Schema

```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    token TEXT,
    role TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## Interface Changes

### `ports.UserRepo`

Updated method signature:
```go
Create(ctx context.Context, name, password, role string) error
```

New methods:
```go
GetAdmin(ctx context.Context) (*User, error)
CountAdmins(ctx context.Context) (int, error)
```

### Error Types

```go
var (
    UserNotFound = errors.New("[no such user]")
    AdminExists  = errors.New("[admin already exists]")
    InvalidRole  = errors.New("[invalid role]")
)
```

### Role Constants

```go
const (
    RoleAdmin = "admin"
    RoleUser  = "user"
)
```

## Behavior

### Creating Users

1. **Admin Creation**
   - Check if admin already exists via `CountAdmins()`
   - If admin exists, return `AdminExists` error
   - Otherwise, create user with role `admin`

2. **User Creation**
   - No restrictions on number of regular users
   - Create user with role `user`

3. **Invalid Role**
   - If role is not `admin` or `user`, return `InvalidRole` error

### Getting Admin

- Returns the single admin user
- Returns `UserNotFound` if no admin exists

## Security

- Role validation on user creation
- Admin uniqueness enforced at database level via application logic
