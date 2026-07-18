package handlers

import (
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/RobMil91/free-orgx/internal/ports"
)

type UserHandler struct {
	Files  fs.FS
	Logger *slog.Logger

	UserRepo ports.UserRepo
}

func NewUserHandler(
	files fs.FS,
	l *slog.Logger,
	u ports.UserRepo,
) *UserHandler {

	return &UserHandler{
		Files:    files,
		Logger:   l,
		UserRepo: u,
	}
}

func (u UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(u.Files, "loginPage.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (u UserHandler) Submit(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(u.Files, "loginSuccessCard.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result, err := u.UserRepo.GetUserToken(r.Context(), r.FormValue("user"), r.FormValue("password"))
	if err != nil {

		tmpl.Execute(w, struct {
			Result string
		}{
			Result: "failed",
		})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieID,
		Value:    result.Cookie.Value,
		HttpOnly: true,
		Path:     "/",
	})

	if result.IsNewUser {
		u.Logger.Info(fmt.Sprintf("first admin registered: %s", result.User.Name))
		return
	}

	tmpl.Execute(w, struct {
		Result string
	}{
		Result: "Success",
	})
}

func (u *UserHandler) AdminUsersPage(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(SessionCookieID)
	if err != nil {
		w.Write([]byte("Please login first"))
		return
	}

	user, err := u.UserRepo.IsValid(r.Context(), c.Value)
	if err != nil {
		w.Write([]byte("Session invalid, please login again"))
		return
	}

	if user.Role != ports.RoleAdmin {
		w.Write([]byte("Access denied: admin only"))
		return
	}

	users, err := u.UserRepo.GetAll(r.Context())
	if err != nil {
		u.Logger.Error("failed to get users", "error", err)
		users = []ports.User{}
	}

	tmpl, err := template.ParseFS(u.Files, "admin_users.html")
	if err != nil {
		u.Logger.ErrorContext(r.Context(), err.Error())
		return

	}
	tmpl.Execute(w, struct {
		Users []ports.User
	}{
		Users: users,
	})
}

func (u *UserHandler) AdminCreateUser(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(SessionCookieID)
	if err != nil {
		w.Write([]byte("Please login first"))
		return
	}

	user, err := u.UserRepo.IsValid(r.Context(), c.Value)
	if err != nil {
		w.Write([]byte("Session invalid, please login again"))
		return
	}

	if user.Role != ports.RoleAdmin {
		w.Write([]byte("Access denied: admin only"))
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		w.Write([]byte("Username and password are required"))
		return
	}

	err = u.UserRepo.Create(r.Context(), username, password, ports.RoleUser)
	if err != nil {
		u.Logger.Error("failed to create user", "error", err)
		w.Write([]byte(fmt.Sprintf("Failed to create user: %s", err.Error())))
		return
	}

	u.Logger.Info(fmt.Sprintf("admin %s created user %s", user.Name, username))

	users, err := u.UserRepo.GetAll(r.Context())
	if err != nil {
		u.Logger.Error("failed to create user list", "error", err)
		return
	}

	tmpl, err := template.ParseFS(u.Files, "userList.html")
	if err != nil {
		u.Logger.Error("failed to creat template", "error", err)
		return
	}

	tmpl.Execute(w, struct {
		Users []ports.User
	}{
		Users: users,
	})

}

func (u *UserHandler) Logout(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(SessionCookieID)
	if err != nil {
		http.Error(w, err.Error(), 401)
		return
	}

	user, err := u.UserRepo.IsValid(r.Context(), c.Value)
	if err != nil {
		http.Error(w, err.Error(), 403)
		return
	}

	if err := u.UserRepo.Logout(r.Context(), user.Name); err != nil {
		u.Logger.Error("logout failed", "error", err)
		http.Error(w, err.Error(), 500)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:   SessionCookieID,
		Value:  "",
		MaxAge: -1,
	})

	http.Redirect(w, r, "/", http.StatusFound)
}
