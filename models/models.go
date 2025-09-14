package models

import (
	"hcs-full/database/db"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserID  uuid.UUID `json:"user_id"`
	Email   string    `json:"email"`
	IsAdmin bool      `json:"is_admin"`
	jwt.RegisteredClaims
}

type PageData struct {
	Title                 string
	IsAuthenticated       bool
	User                  *db.User
	UserAppointments      []db.Appointment
	AllAppointments       []db.GetAllAppointmentsRow
	Success               string
	Error                 string
	ErrorMessage          string
	TotalAppointments     int
	AcceptedAppointments  int
	PendingAppointments   int
	CancelledAppointments int
	Calendar              CalendarData
	ChartData             ChartData
}

type CalendarData struct {
	Month       string
	Year        int
	DaysOfWeek  []string
	Days        []CalendarDay
	MonthIndex  int
}

type CalendarDay struct {
	Number       int
	IsToday      bool
	Appointments []AppointmentInfo
}

type AppointmentInfo struct {
	Title string `json:"title"`
	Time  string `json:"time"`
}

// ChartData holds the monthly appointment counts for the overview chart.
type ChartData struct {
	Labels    []string `json:"labels"`
	Confirmed []int    `json:"confirmed"`
	Pending   []int    `json:"pending"`
	Cancelled []int    `json:"cancelled"`
}

// Message is the structure for WebSocket messages
type Message struct {
	UserID uuid.UUID       `json:"userId"`
	Type   string          `json:"type"`
	Data   interface{}     `json:"data"`
}


