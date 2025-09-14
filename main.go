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
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Connect to the database
	database.ConnectDB()
	defer database.CloseDB()

	// Run database migrations
	if err := database.RunMigrations(); err != nil {
		log.Fatalf("Could not run database migrations: %v", err)
	}

	// Seed the initial admin user
	if err := database.SeedAdminUser(); err != nil {
		log.Fatalf("Could not seed admin user: %v", err)
	}

	// Initialize WebSocket Hub
	handlers.WsHub = handlers.NewHub()
	go handlers.WsHub.Run()

	// Static file server
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
	http.Handle("/delete-appointment", middleware.AuthMiddleware(http.HandlerFunc(handlers.DeleteAppointmentHandler)))

	// WebSocket route
	http.Handle("/ws", middleware.SoftAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlers.ServeWs(handlers.WsHub, w, r)
	})))

	// Admin routes
	adminRoutes := http.NewServeMux()
	adminRoutes.Handle("/admin/dashboard", http.HandlerFunc(handlers.AdminDashboardHandler))
	adminRoutes.Handle("/admin/update-status", http.HandlerFunc(handlers.AdminUpdateAppointmentStatusHandler))
	http.Handle("/admin/", middleware.AuthMiddleware(middleware.AdminMiddleware(adminRoutes)))

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on :%s...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}


