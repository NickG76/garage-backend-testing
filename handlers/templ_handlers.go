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

// AboutPageTempl renders the about page using templ
func AboutPageTempl(w http.ResponseWriter, r *http.Request) {
	RenderTemplPage(w, r, templPkg.AboutPage, models.PageData{Title: "About Us"})
}

// ServicesPageTempl renders the services page using templ
func ServicesPageTempl(w http.ResponseWriter, r *http.Request) {
	RenderTemplPage(w, r, templPkg.ServicesPage, models.PageData{Title: "Our Services"})
}

// ContactPageTempl renders the contact page using templ
func ContactPageTempl(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		RenderTemplPage(w, r, templPkg.ContactPage, models.PageData{Title: "Contact Us"})
		return
	}

	// Handle POST request (contact form submission)
	name := r.FormValue("name")
	email := r.FormValue("email")
	_ = r.FormValue("phone") // phone is optional
	subject := r.FormValue("subject")
	message := r.FormValue("message")

	// Basic validation
	if name == "" || email == "" || subject == "" || message == "" {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<div class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded relative mb-4" role="alert">
			<strong class="font-bold">Error: </strong>
			<span class="block sm:inline">Please fill in all required fields.</span>
		</div>`))
		return
	}

	// Here you would typically send an email or save to database
	// For now, just return a success message
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<div class="bg-green-100 border border-green-400 text-green-700 px-4 py-3 rounded relative mb-4" role="alert">
		<strong class="font-bold">Success! </strong>
		<span class="block sm:inline">Thank you for your message. We'll get back to you within 24 hours.</span>
	</div>`))
}
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