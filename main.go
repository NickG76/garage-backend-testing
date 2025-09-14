package main

import (
	"log"
	"net/http"
	"os"

	"hcs-full/database"
	"hcs-full/handlers"
	"hcs-full/middleware"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	database.ConnectDB()
	defer database.CloseDB()

	if err := database.RunMigrations(); err != nil {
		log.Fatalf("Could not run database migrations: %v", err)
	}

	if err := database.SeedAdminUser(); err != nil {
		log.Fatalf("Could not seed admin user: %v", err)
	}

	handlers.WsHub = handlers.NewHub()
	go handlers.WsHub.Run()

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Public routes
	http.HandleFunc("/", handlers.HomePage)
	http.HandleFunc("/about", handlers.AboutPage)
	http.HandleFunc("/services", handlers.ServicesPage)
	http.HandleFunc("/contact", handlers.ContactPage)
	http.HandleFunc("/privacy-policy", handlers.PrivacyPolicyPage)
	http.HandleFunc("/terms-and-conditions", handlers.TermsAndConditionsPage)

	// Auth routes
	http.HandleFunc("/login", handlers.LoginHandler)
	http.HandleFunc("/signup", handlers.SignupHandler)
	http.HandleFunc("/logout", handlers.LogoutHandler)

	// Authenticated routes
	http.Handle("/dashboard", middleware.AuthMiddleware(http.HandlerFunc(handlers.DashboardHandler)))
	http.Handle("/create-appointment", middleware.AuthMiddleware(http.HandlerFunc(handlers.CreateAppointmentHandler)))
	http.Handle("/cancel-appointment", middleware.AuthMiddleware(http.HandlerFunc(handlers.UserCancelAppointmentHandler)))

	// WebSocket route
	http.Handle("/ws", middleware.SoftAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlers.ServeWs(handlers.WsHub, w, r)
	})))

	// Admin routes
	adminRoutes := http.NewServeMux()
	adminRoutes.Handle("/admin/overview", http.HandlerFunc(handlers.AdminOverviewHandler))
	adminRoutes.Handle("/admin/dashboard", http.HandlerFunc(handlers.AdminDashboardHandler))
	adminRoutes.Handle("/admin/update-status", http.HandlerFunc(handlers.AdminUpdateAppointmentStatusHandler))
	adminRoutes.Handle("/admin/delete-appointment", http.HandlerFunc(handlers.AdminDeleteAppointmentHandler))
	adminRoutes.Handle("/api/admin/calendar", http.HandlerFunc(handlers.AdminCalendarHandler))
	http.Handle("/admin/", middleware.AuthMiddleware(middleware.AdminMiddleware(adminRoutes)))
	http.Handle("/api/admin/", middleware.AuthMiddleware(middleware.AdminMiddleware(adminRoutes)))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on :%s...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}


