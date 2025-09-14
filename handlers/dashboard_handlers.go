package handlers

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"hcs-full/database"
	"hcs-full/database/db"
	"hcs-full/models"
	"hcs-full/utils"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// generateChartData calculates appointment statistics for the last 12 months.
func generateChartData(appointments []db.Appointment) models.ChartData {
	now := time.Now()
	chartData := models.ChartData{
		Labels:    make([]string, 12),
		Confirmed: make([]int, 12),
		Pending:   make([]int, 12),
		Cancelled: make([]int, 12),
	}

	for i := 0; i < 12; i++ {
		// Go backwards in time month by month
		month := now.AddDate(0, -i, 0)
		// Build labels array from oldest to newest
		chartData.Labels[11-i] = month.Format("Jan")

		// Iterate through all appointments to count stats for the current month in the loop
		for _, appt := range appointments {
			if appt.Datetime.Time.Year() == month.Year() && appt.Datetime.Time.Month() == month.Month() {
				switch appt.Status {
				case "confirmed", "completed":
					chartData.Confirmed[11-i]++
				case "pending":
					chartData.Pending[11-i]++
				case "cancelled":
					chartData.Cancelled[11-i]++
				}
			}
		}
	}

	return chartData
}

func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value("userClaims").(*models.Claims)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	user, err := database.Queries.GetUserByID(context.Background(), pgtype.UUID{Bytes: claims.UserID, Valid: true})
	if err != nil {
		RenderTemplate(w, r, "error.html", models.PageData{Title: "Error", ErrorMessage: "User not found."})
		return
	}

	appointments, err := database.Queries.GetAppointmentsForUser(context.Background(), pgtype.UUID{Bytes: claims.UserID, Valid: true})
	if err != nil {
		log.Printf("Error getting appointments for user %s: %v", claims.UserID, err)
		RenderTemplate(w, r, "error.html", models.PageData{Title: "Error", ErrorMessage: "Could not retrieve appointments."})
		return
	}

	// Calculate stats for cards
	var total, accepted, pending, cancelled int
	for _, appt := range appointments {
		total++
		switch appt.Status {
		case "confirmed", "completed":
			accepted++
		case "pending":
			pending++
		case "cancelled":
			cancelled++
		}
	}

	// Generate calendar data for the current month
	now := time.Now()
	calendarData := generateCalendarData(now, appointments)

	// Generate chart data for the last 12 months
	chartData := generateChartData(appointments)

	data := models.PageData{
		Title:                 "Dashboard",
		IsAuthenticated:       true,
		User:                  &user,
		UserAppointments:      appointments,
		TotalAppointments:     total,
		AcceptedAppointments:  accepted,
		PendingAppointments:   pending,
		CancelledAppointments: cancelled,
		Calendar:              calendarData,
		ChartData:             chartData,
	}

	RenderTemplate(w, r, "dashboard.html", data)
}

func CreateAppointmentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}

	claims, ok := r.Context().Value("userClaims").(*models.Claims)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	datetimeStr := r.FormValue("datetime")
	datetime, err := time.Parse("2006-01-02T15:04", datetimeStr)
	if err != nil {
		http.Error(w, "Invalid date format", http.StatusBadRequest)
		return
	}

	params := db.CreateAppointmentParams{
		UserID:      pgtype.UUID{Bytes: claims.UserID, Valid: true},
		Datetime:    pgtype.Timestamptz{Time: datetime, Valid: true},
		Title:       utils.SanitizeInput(r.FormValue("title")),
		Description: pgtype.Text{String: utils.SanitizeInput(r.FormValue("description")), Valid: true},
	}

	appt, err := database.Queries.CreateAppointment(context.Background(), params)
	if err != nil {
		log.Printf("Error creating appointment: %v", err)
		http.Error(w, "Error creating appointment", http.StatusInternalServerError)
		return
	}

	// Notify admins via WebSocket
	notification := map[string]interface{}{
		"type": "new_appointment",
		"data": map[string]string{
			"appointmentId": hex.EncodeToString(appt.ID.Bytes[:]),
			"title":         appt.Title,
			"userName":      claims.Email, // or User's name if you fetch it
		},
	}
	jsonMsg, _ := json.Marshal(notification)
	WsHub.broadcast <- jsonMsg

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func DeleteAppointmentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}

	appointmentIDStr := r.FormValue("id")
	idBytes, err := hex.DecodeString(appointmentIDStr)
	if err != nil || len(idBytes) != 16 {
		http.Error(w, "Invalid appointment ID", http.StatusBadRequest)
		return
	}

	var pgUUID pgtype.UUID
	copy(pgUUID.Bytes[:], idBytes)
	pgUUID.Valid = true

	err = database.Queries.DeleteAppointment(context.Background(), pgUUID)
	if err != nil {
		log.Printf("Error deleting appointment: %v", err)
		http.Error(w, "Error deleting appointment", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func AdminDashboardHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value("userClaims").(*models.Claims)
	if !ok || !claims.IsAdmin {
		RenderTemplate(w, r, "error.html", models.PageData{Title: "Forbidden", ErrorMessage: "You do not have permission to view this page."})
		return
	}

	allAppointments, err := database.Queries.GetAllAppointments(context.Background())
	if err != nil {
		log.Printf("Error getting all appointments: %v", err)
	}

	data := models.PageData{
		Title:           "Admin Dashboard",
		IsAuthenticated: true,
		User: &db.User{
			Name:    claims.Email,
			IsAdmin: true,
		},
		AllAppointments: allAppointments,
	}

	RenderTemplate(w, r, "admin_dashboard.html", data)
}

func AdminUpdateAppointmentStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
		return
	}

	appointmentIDStr := r.FormValue("id")
	idBytes, err := hex.DecodeString(appointmentIDStr)
	if err != nil || len(idBytes) != 16 {
		http.Error(w, "Invalid appointment ID", http.StatusBadRequest)
		return
	}

	var pgUUID pgtype.UUID
	copy(pgUUID.Bytes[:], idBytes)
	pgUUID.Valid = true

	status := r.FormValue("status")

	params := db.UpdateAppointmentStatusParams{
		ID:     pgUUID,
		Status: status,
	}

	err = database.Queries.UpdateAppointmentStatus(context.Background(), params)
	if err != nil {
		log.Printf("Error updating appointment status: %v", err)
		http.Error(w, "Error updating status", http.StatusInternalServerError)
		return
	}

	// Fetch appointment details to get UserID and Title for notification
	appt, err := database.Queries.GetAppointmentsByID(context.Background(), pgUUID)
	if err != nil {
		log.Printf("Could not fetch appointment for notification: %v", err)
	} else {
		// Notify the specific user via WebSocket
		unicastMessage := &models.Message{
			UserID: appt.UserID.Bytes,
			Type:   "status_update",
			Data: map[string]interface{}{
				"appointmentId": hex.EncodeToString(appt.ID.Bytes[:]),
				"title":         appt.Title,
				"status":        appt.Status,
			},
		}
		WsHub.unicast <- unicastMessage
	}

	http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
}

func generateCalendarData(t time.Time, appointments []db.Appointment) models.CalendarData {
	year, month, _ := t.Date()
	firstDay := time.Date(year, month, 1, 0, 0, 0, 0, t.Location())
	lastDay := firstDay.AddDate(0, 1, -1)

	appointmentsMap := make(map[int][]models.AppointmentInfo)
	for _, appt := range appointments {
		if appt.Datetime.Time.Year() == year && appt.Datetime.Time.Month() == month {
			day := appt.Datetime.Time.Day()
			appointmentsMap[day] = append(appointmentsMap[day], models.AppointmentInfo{
				Title: appt.Title,
				Time:  appt.Datetime.Time.Format("3:04 PM"),
			})
		}
	}

	var days []models.CalendarDay
	// Add blank days for the first week
	for i := 0; i < int(firstDay.Weekday()); i++ {
		days = append(days, models.CalendarDay{Number: 0})
	}

	// Add days of the month
	for day := 1; day <= lastDay.Day(); day++ {
		days = append(days, models.CalendarDay{
			Number:       day,
			IsToday:      day == t.Day() && month == t.Month() && year == t.Year(),
			Appointments: appointmentsMap[day],
		})
	}

	return models.CalendarData{
		Month:       t.Format("January"),
		Year:        year,
		DaysOfWeek:  []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"},
		Days:        days,
		MonthIndex: int(month),
	}
}


