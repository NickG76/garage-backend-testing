package handlers

import (
	"context"
	"hcs-full/database"
	"hcs-full/models"
	templPkg "hcs-full/templates_templ"
	"hcs-full/utils"
	"net/http"

	"github.com/a-h/templ"
	"github.com/jackc/pgx/v5/pgtype"
)

// RenderTemplPage renders a templ-based page with authentication check
func RenderTemplPage(w http.ResponseWriter, r *http.Request, component func(models.PageData) templ.Component, data models.PageData) {
	// Check for authentication
	c, err := r.Cookie("token")
	if err == nil {
		claims, err := utils.ParseJWT(c.Value)
		if err == nil {
			data.IsAuthenticated = true
			// If the handler didn't already provide user data, fetch it
			if data.User == nil {
				user, err := database.Queries.GetUserByID(context.Background(), pgtype.UUID{Bytes: claims.UserID, Valid: true})
				if err == nil {
					data.User = &user
				}
			}
		}
	}

	// Render the templ component
	component(data).Render(context.Background(), w)
}

// HomePageTempl renders the home page using templ
func HomePageTempl(w http.ResponseWriter, r *http.Request) {
	RenderTemplPage(w, r, templPkg.IndexPage, models.PageData{Title: "Home"})
}

// LoginPageTempl renders the login page using templ
func LoginPageTempl(w http.ResponseWriter, r *http.Request) {
	data := models.PageData{Title: "Login"}
	
	// Check for error or success messages from query parameters
	if err := r.URL.Query().Get("error"); err != "" {
		data.Error = err
	}
	if success := r.URL.Query().Get("success"); success != "" {
		data.Success = success
	}
	
	RenderTemplPage(w, r, templPkg.LoginPage, data)
}

// LoginHandlerTempl handles login form submission with HTMX
func LoginHandlerTempl(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		LoginPageTempl(w, r)
		return
	}

	// Handle POST request (form submission)
	email := r.FormValue("email")
	password := r.FormValue("password")

	if email == "" || password == "" {
		data := models.PageData{
			Title: "Login",
			Error: "Email and password are required",
		}
		RenderTemplPage(w, r, templPkg.LoginPage, data)
		return
	}

	user, err := database.Queries.GetUserByEmail(context.Background(), email)
	if err != nil {
		data := models.PageData{
			Title: "Login",
			Error: "Invalid email or password",
		}
		RenderTemplPage(w, r, templPkg.LoginPage, data)
		return
	}

	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		data := models.PageData{
			Title: "Login",
			Error: "Invalid email or password",
		}
		RenderTemplPage(w, r, templPkg.LoginPage, data)
		return
	}

	// Generate JWT token
	token, err := utils.GenerateJWT(user.ID.Bytes, user.Email, user.IsAdmin)
	if err != nil {
		data := models.PageData{
			Title: "Login",
			Error: "Failed to generate authentication token",
		}
		RenderTemplPage(w, r, templPkg.LoginPage, data)
		return
	}

	// Set the token as a cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		MaxAge:   86400, // 24 hours
	})

	// For HTMX, we need to trigger a redirect
	w.Header().Set("HX-Redirect", "/dashboard")
	w.WriteHeader(http.StatusOK)
}

// LogoutHandlerTempl handles logout with HTMX
func LogoutHandlerTempl(w http.ResponseWriter, r *http.Request) {
	// Clear the token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		MaxAge:   -1, // Delete the cookie
	})

	// For HTMX, redirect to home page
	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusOK)
}