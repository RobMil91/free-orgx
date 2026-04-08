# Admin User Management Feature Specification

## Overview

The admin has exclusive access to user management functionality. This allows the admin to create new users (with `user` role) in the system.

## Access Control

| Endpoint | Access |
|----------|--------|
| `/admin/users` | Admin only |
| `/admin/user/create` | Admin only |

Non-admin users and unauthenticated requests receive "Access denied" or "Please login first" respectively.

## Endpoints

### GET `/admin/users`

Displays a list of all users in the system.

**Authentication:** Required (session cookie)

**Authorization:** Admin role only

**Response:**
- Success: HTML page listing all users with their roles
- Unauthenticated: "Please login first"
- Unauthorized: "Access denied: admin only"

### POST `/admin/user/create`

Creates a new user with `user` role.

**Authentication:** Required (session cookie)

**Authorization:** Admin role only

**Request Body:**
```
username=<name>&password=<password>
```

**Response:**
- Success: "User '<name>' created successfully"
- Missing fields: "Username and password are required"
- Unauthenticated: "Please login first"
- Unauthorized: "Access denied: admin only"
- Error: "Failed to create user: <error>"

## UI Flow

1. Admin logs in
2. Admin sees "Manage Users" button on project page
3. Admin clicks button to view user list
4. Admin fills form to create new user
5. New user can now login with credentials

## Template Data

### Project Page (`project.html`)

Added `IsAdmin` field:
```go
struct {
    Username string
    Projects []models.Project
    IsAdmin  bool
}
```

### Admin Users Page (`admin_users.html`)

```go
struct {
    Users []ports.User
}
```

## Security

- Admin-only endpoints validated server-side
- Role check on every request
- Logging of user creation events

## Error Handling

| Scenario | Response |
|----------|----------|
| No cookie | "Please login first" |
| Invalid session | "Session invalid, please login again" |
| Non-admin user | "Access denied: admin only" |
| Missing username/password | "Username and password are required" |
| Duplicate user | Handled by Create() - returns error |
