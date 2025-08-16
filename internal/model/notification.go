package model

import (
	"time"

	"github.com/google/uuid"
)

// Notification representa una notificación en el sistema
type Notification struct {
	ID         uuid.UUID              `json:"id" db:"id"`
	Type       NotificationType       `json:"type" db:"type"`
	Status     NotificationStatus     `json:"status" db:"status"`
	Priority   NotificationPriority   `json:"priority" db:"priority"`
	Recipient  string                 `json:"recipient" db:"recipient"`
	Subject    string                 `json:"subject" db:"subject"`
	Content    string                 `json:"content" db:"content"`
	TemplateID string                 `json:"template_id" db:"template_id"`
	Data       map[string]interface{} `json:"data" db:"data"`
	SentAt     *time.Time             `json:"sent_at" db:"sent_at"`
	ReadAt     *time.Time             `json:"read_at" db:"read_at"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at" db:"updated_at"`
}

// NotificationType define los tipos de notificaciones
type NotificationType string

const (
	NotificationTypeEventCreated         NotificationType = "event_created"
	NotificationTypeEventUpdated         NotificationType = "event_updated"
	NotificationTypeEventCancelled       NotificationType = "event_cancelled"
	NotificationTypeEventReminder        NotificationType = "event_reminder"
	NotificationTypeReservationCreated   NotificationType = "reservation_created"
	NotificationTypeReservationConfirmed NotificationType = "reservation_confirmed"
	NotificationTypeReservationCancelled NotificationType = "reservation_cancelled"
	NotificationTypeTicketGenerated      NotificationType = "ticket_generated"
	NotificationTypePaymentReceived      NotificationType = "payment_received"
	NotificationTypePaymentFailed        NotificationType = "payment_failed"
	NotificationTypeWelcome              NotificationType = "welcome"
	NotificationTypePasswordReset        NotificationType = "password_reset"
)

// NotificationStatus define el estado de una notificación
type NotificationStatus string

const (
	NotificationStatusPending   NotificationStatus = "pending"
	NotificationStatusSending   NotificationStatus = "sending"
	NotificationStatusSent      NotificationStatus = "sent"
	NotificationStatusDelivered NotificationStatus = "delivered"
	NotificationStatusFailed    NotificationStatus = "failed"
	NotificationStatusRead      NotificationStatus = "read"
)

// NotificationPriority define la prioridad de una notificación
type NotificationPriority string

const (
	NotificationPriorityLow    NotificationPriority = "low"
	NotificationPriorityNormal NotificationPriority = "normal"
	NotificationPriorityHigh   NotificationPriority = "high"
	NotificationPriorityUrgent NotificationPriority = "urgent"
)

// CreateNotificationRequest representa la solicitud para crear una notificación
type CreateNotificationRequest struct {
	Type       NotificationType       `json:"type" binding:"required"`
	Priority   NotificationPriority   `json:"priority"`
	Recipient  string                 `json:"recipient" binding:"required"`
	Subject    string                 `json:"subject" binding:"required"`
	Content    string                 `json:"content" binding:"required"`
	TemplateID string                 `json:"template_id"`
	Data       map[string]interface{} `json:"data"`
}

// UpdateNotificationRequest representa la solicitud para actualizar una notificación
type UpdateNotificationRequest struct {
	Status *NotificationStatus `json:"status"`
	SentAt *time.Time          `json:"sent_at"`
	ReadAt *time.Time          `json:"read_at"`
}

// NotificationTemplate representa una plantilla de notificación
type NotificationTemplate struct {
	ID        uuid.UUID        `json:"id" db:"id"`
	Name      string           `json:"name" db:"name"`
	Type      NotificationType `json:"type" db:"type"`
	Subject   string           `json:"subject" db:"subject"`
	Content   string           `json:"content" db:"content"`
	Variables []string         `json:"variables" db:"variables"`
	IsActive  bool             `json:"is_active" db:"is_active"`
	CreatedAt time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt time.Time        `json:"updated_at" db:"updated_at"`
}

// EventNotification representa una notificación específica de evento
type EventNotification struct {
	EventID   string               `json:"event_id" binding:"required"`
	EventName string               `json:"event_name" binding:"required"`
	EventDate time.Time            `json:"event_date" binding:"required"`
	Location  string               `json:"location" binding:"required"`
	Recipient string               `json:"recipient" binding:"required"`
	Type      NotificationType     `json:"type" binding:"required"`
	Priority  NotificationPriority `json:"priority"`
}

// ReservationNotification representa una notificación específica de reserva
type ReservationNotification struct {
	ReservationID string               `json:"reservation_id" binding:"required"`
	EventID       string               `json:"event_id" binding:"required"`
	EventName     string               `json:"event_name" binding:"required"`
	EventDate     time.Time            `json:"event_date" binding:"required"`
	Location      string               `json:"location" binding:"required"`
	Recipient     string               `json:"recipient" binding:"required"`
	Type          NotificationType     `json:"type" binding:"required"`
	Priority      NotificationPriority `json:"priority"`
}

// BulkNotificationRequest representa una solicitud para enviar múltiples notificaciones
type BulkNotificationRequest struct {
	Notifications []CreateNotificationRequest `json:"notifications" binding:"required"`
	TemplateID    string                      `json:"template_id"`
	Priority      NotificationPriority        `json:"priority"`
}

