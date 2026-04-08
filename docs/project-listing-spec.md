# Project Listing Feature Specification

## Overview

The project listing feature displays all projects owned by the currently authenticated user. Projects are filtered by owner to ensure users only see their own projects.

## Functionality

Projects hold an ammount of tasks.
Tasks are displayed as a list with the following details:
- Task ID
- Title
- Description
- Deadline
- Status (e.g., "In Progress", "Done", Blocked)
- Members (e.g., "John Doe", "Jane Smith")
- Owner

### Core Features

1. **List User Projects**
   - Fetch all projects where `current_user = part of the project`
   - Display project name and ID for each project
   - Show empty state message if user has no projects

2. **Authentication Required**
   - Users must be logged in to view their projects
   - Redirect to login prompt if not authenticated

3. Create New Projects

3. **Delete Projects**
    - Owners can delete Project

### Data Flow

1. User navigates to `/project`
2. Server validates session cookie
3. Server queries projects filtered by authenticated user's username
4. Server renders projects list in HTML template

### API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/project` | GET | List user's projects |

### Database Schema


#### `ports.ProjectRepo`

Added new method:
```go
GetProjectsByOwner(ctx context.Context, owner string) ([]models.Project, error)
```

Modified method:
```go
// CreateProject now requires owner parameter
CreateProject(ctx context.Context, name, owner string) (models.Project, error)
```

## User Interface

### Project Page (`project.html`)

- Displays username heading
- Shows project creation form
- Lists all user's projects in a `<ul>`
- Empty state: "No projects yet. Create one above!"

### Template Data

```go
struct {
    Username string
    Projects []models.Project
}
```

Projects can be renamed and delted. they organizse around the id.