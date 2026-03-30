# First-User-Is-Admin Feature Specification

## Overview

On a fresh installation (empty database), the first user to log in automatically becomes the admin. This eliminates the need for manual admin creation during setup.

## Behavior

### First Login (Empty Database)

1. User attempts to login with a new username/password
2. System detects no users exist
3. System creates user with `admin` role automatically
4. User receives session cookie and is logged in
5. Login response indicates this was a new admin registration

### Subsequent Logins (Users Exist)

1. User attempts to login with non-existent username
2. System detects users already exist
3. System returns `UserNotFound` error
4. User cannot self-register; admin must create accounts

### Existing Users

1. User attempts to login with existing username and correct password
2. System authenticates successfully
3. User receives session cookie and is logged in

## Interface Changes

### `LoginResult` Type

```go
type LoginResult struct {
    Cookie    *SessionCookie
    User      *User
    IsNewUser bool
}
```

### `UserRepo.GetUserToken` Signature

```go
GetUserToken(ctx context.Context, name, password string) (*LoginResult, error)
```

Returns `LoginResult` instead of just `SessionCookie` to provide:
- Session cookie for authentication
- User info including role
- Flag indicating if this was a new user registration

### New `CountUsers` Method

```go
CountUsers(ctx context.Context) (int, error)
```

Returns total count of users in the system.

## Login Flow

```
┌─────────────────┐
│  Login Request  │
│  (name, pass)   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  User exists?   │
└────────┬────────┘
         │
    Yes  │  No
    ┌────┴────┐
    │         │
    ▼         ▼
┌───────┐  ┌─────────────────┐
│ Check │  │ Users = 0?     │
│ creds │  └────────┬────────┘
└───┬───┘           │
    │          Yes  │  No
    │          ┌────┴────┐
    │          │         │
    ▼          ▼         ▼
┌───────┐  ┌────────┐ ┌──────────────┐
│ OK    │  │Create  │ │Return        │
│ cookie│  │admin   │ │UserNotFound  │
└───────┘  └────────┘ └──────────────┘
```

## Database Initialization

On fresh start:
- Database file created with schema
- No default admin user pre-created
- First login becomes admin automatically

## Security Considerations

- No self-registration after first user
- Admin must create additional users via separate mechanism
- Single admin constraint still enforced

## Error Handling

| Scenario | Error |
|----------|-------|
| First login (no users) | Success, user created as admin |
| Valid credentials | Success, session cookie returned |
| Wrong password | `bad password` |
| Non-existent user (users exist) | `UserNotFound` |
