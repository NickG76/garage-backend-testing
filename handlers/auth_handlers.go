package handlers

import (
	"context"
	"database/sql"
	"hcs-full/database"
	"hcs-full/database/db"
	"hcs-full/models"
	"hcs-full/utils"
	"log"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)


func SignupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		name := utils.SanitizeInput(r.FormValue("name"))
		email := utils.SanitizeInput(r.FormValue("email"))
		password := r.FormValue("password")
		phone := utils.SanitizeInput(r.FormValue("phone"))

		if len(password) < 8 {
			RenderTemplate(w, r, "signup.html", models.PageData{Title: "Sign Up", Error: "Password must be at least 8 characters long."})
			return
		}

		hashedPassword, err := utils.HashPassword(password)
		if err != nil {
			http.Error(w, "Server error, unable to hash password.", http.StatusInternalServerError)
			return
		}

		params := db.CreateUserParams{
			Name:         name,
			Email:        email,
			PasswordHash: hashedPassword,
			Phone:        phone,
			IsAdmin:      false,
		}

		_, err = database.Queries.CreateUser(context.Background(), params)
		if err != nil {
			log.Printf("Could not create user: %v", err)
			RenderTemplate(w, r, "signup.html", models.PageData{Title: "Sign Up", Error: "Email already exists."})
			return
		}

		http.Redirect(w, r, "/login?success=true", http.StatusSeeOther)
		return
	}

	RenderTemplate(w, r, "signup.html", models.PageData{Title: "Sign Up"})
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		email := utils.SanitizeInput(r.FormValue("email"))
		password := r.FormValue("password")

		user, err := database.Queries.GetUserByEmail(context.Background(), email)
		if err != nil {
			if err == sql.ErrNoRows {
				RenderTemplate(w, r, "login.html", models.PageData{Title: "Login", Error: "Invalid email or password."})
			} else {
				http.Error(w, "Database error", http.StatusInternalServerError)
			}
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
		if err != nil {
			RenderTemplate(w, r, "login.html", models.PageData{Title: "Login", Error: "Invalid email or password."})
			return
		}

		token, err := utils.GenerateJWT(user.ID.Bytes, user.Email, user.IsAdmin)
		if err != nil {
			http.Error(w, "Could not generate token", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    token,
			Expires:  time.Now().Add(24 * time.Hour),
			HttpOnly: true,
			Path:     "/",
			SameSite: http.SameSiteStrictMode,
		})

		if user.IsAdmin {
			http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
		} else {
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		}
		return
	}

	success := r.URL.Query().Get("success")
	data := models.PageData{Title: "Login"}
	if success == "true" {
		data.Success = "Registration successful! Please log in."
	}
	RenderTemplate(w, r, "login.html", data)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour), // Set expiry to the past
		HttpOnly: true,
		Path:     "/",
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}


