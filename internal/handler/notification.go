package handler

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jhonathanssegura/ticket-notification/internal/db"
	"github.com/jhonathanssegura/ticket-notification/internal/model"
	"github.com/jhonathanssegura/ticket-notification/internal/service"
)

// NotificationHandler maneja las peticiones HTTP relacionadas con notificaciones
type NotificationHandler struct {
	notificationService *service.NotificationService
	dbClient            *db.DynamoClient
}

// NewNotificationHandler crea una nueva instancia del handler de notificaciones
func NewNotificationHandler(notificationService *service.NotificationService, dbClient *db.DynamoClient) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
		dbClient:            dbClient,
	}
}

// SendNotification envía una notificación individual
func (h *NotificationHandler) SendNotification(c *gin.Context) {
	var req model.CreateNotificationRequest

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Datos de notificación inválidos",
			"details": err.Error(),
		})
		return
	}

	// Validar campos requeridos
	if req.Recipient == "" || req.Subject == "" || req.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Recipient, subject y content son campos requeridos",
		})
		return
	}

	// Enviar notificación
	notification, err := h.notificationService.SendNotification(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error enviando notificación",
			"details": err.Error(),
		})
		return
	}

	// Guardar en base de datos
	if err := h.dbClient.SaveNotification(*notification); err != nil {
		log.Printf("Error guardando notificación en DB: %v", err)
		// No fallar la request si solo falla el guardado en DB
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    notification,
		"message": "Notificación enviada exitosamente",
	})
}

// SendBulkNotifications envía múltiples notificaciones
func (h *NotificationHandler) SendBulkNotifications(c *gin.Context) {
	var req model.BulkNotificationRequest

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Datos de notificaciones inválidos",
			"details": err.Error(),
		})
		return
	}

	if len(req.Notifications) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Debe especificar al menos una notificación",
		})
		return
	}

	// Enviar notificaciones en lote
	notifications, err := h.notificationService.SendBulkNotifications(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error enviando notificaciones en lote",
			"details": err.Error(),
		})
		return
	}

	// Guardar en base de datos
	for _, notification := range notifications {
		if err := h.dbClient.SaveNotification(*notification); err != nil {
			log.Printf("Error guardando notificación %s en DB: %v", notification.ID, err)
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"notifications":   notifications,
			"total_sent":      len(notifications),
			"total_requested": len(req.Notifications),
		},
		"message": "Notificaciones en lote enviadas exitosamente",
	})
}

// GetNotification obtiene una notificación por ID
func (h *NotificationHandler) GetNotification(c *gin.Context) {
	notificationID := c.Param("id")
	if notificationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de notificación requerido"})
		return
	}

	notification, err := h.dbClient.GetNotificationByID(notificationID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Notificación no encontrada"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error obteniendo notificación",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    notification,
	})
}

// ListNotifications lista notificaciones con filtros opcionales
func (h *NotificationHandler) ListNotifications(c *gin.Context) {
	recipient := c.Query("recipient")
	notificationType := c.Query("type")
	limitStr := c.Query("limit")

	limit := 50 // límite por defecto
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	notifications, err := h.dbClient.GetNotifications(recipient, notificationType, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error obteniendo notificaciones",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"notifications": notifications,
			"count":         len(notifications),
			"limit":         limit,
			"filters": gin.H{
				"recipient": recipient,
				"type":      notificationType,
			},
		},
	})
}

// UpdateNotification actualiza una notificación existente
func (h *NotificationHandler) UpdateNotification(c *gin.Context) {
	notificationID := c.Param("id")
	if notificationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de notificación requerido"})
		return
	}

	var req model.UpdateNotificationRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Datos de actualización inválidos",
			"details": err.Error(),
		})
		return
	}

	// Preparar actualizaciones
	updates := make(map[string]interface{})
	if req.Status != nil {
		updates["status"] = string(*req.Status)
	}
	if req.SentAt != nil {
		updates["sent_at"] = *req.SentAt
	}
	if req.ReadAt != nil {
		updates["read_at"] = *req.ReadAt
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Debe especificar al menos un campo para actualizar"})
		return
	}

	// Actualizar en base de datos
	if err := h.dbClient.UpdateNotification(notificationID, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error actualizando notificación",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notificación actualizada exitosamente",
	})
}

// DeleteNotification elimina una notificación
func (h *NotificationHandler) DeleteNotification(c *gin.Context) {
	notificationID := c.Param("id")
	if notificationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de notificación requerido"})
		return
	}

	if err := h.dbClient.DeleteNotification(notificationID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error eliminando notificación",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notificación eliminada exitosamente",
	})
}

// NotifyEventCreated notifica cuando se crea un evento
func (h *NotificationHandler) NotifyEventCreated(c *gin.Context) {
	var req model.EventNotification

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Datos de notificación de evento inválidos",
			"details": err.Error(),
		})
		return
	}

	// Validar campos requeridos
	if req.EventID == "" || req.EventName == "" || req.Recipient == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "EventID, EventName y Recipient son campos requeridos",
		})
		return
	}

	// Si no se especifica prioridad, usar normal
	if req.Priority == "" {
		req.Priority = model.NotificationPriorityNormal
	}

	// Enviar notificación
	if err := h.notificationService.NotifyEventCreated(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error enviando notificación de evento",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notificación de evento enviada exitosamente",
		"data": gin.H{
			"event_id":   req.EventID,
			"event_name": req.EventName,
			"recipient":  req.Recipient,
			"type":       req.Type,
		},
	})
}

// SendEventReminder envía un recordatorio de evento
func (h *NotificationHandler) SendEventReminder(c *gin.Context) {
	var req model.EventNotification

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Datos de recordatorio de evento inválidos",
			"details": err.Error(),
		})
		return
	}

	// Validar campos requeridos
	if req.EventID == "" || req.EventName == "" || req.Recipient == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "EventID, EventName y Recipient son campos requeridos",
		})
		return
	}

	// Enviar recordatorio
	if err := h.notificationService.SendEventReminder(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error enviando recordatorio de evento",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Recordatorio de evento enviado exitosamente",
		"data": gin.H{
			"event_id":   req.EventID,
			"event_name": req.EventName,
			"recipient":  req.Recipient,
		},
	})
}

// NotifyEventCancelled notifica cuando se cancela un evento
func (h *NotificationHandler) NotifyEventCancelled(c *gin.Context) {
	var req model.EventNotification

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Datos de cancelación de evento inválidos",
			"details": err.Error(),
		})
		return
	}

	// Validar campos requeridos
	if req.EventID == "" || req.EventName == "" || req.Recipient == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "EventID, EventName y Recipient son campos requeridos",
		})
		return
	}

	// Enviar notificación de cancelación
	if err := h.notificationService.NotifyEventCancelled(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error enviando notificación de cancelación de evento",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notificación de cancelación de evento enviada exitosamente",
		"data": gin.H{
			"event_id":   req.EventID,
			"event_name": req.EventName,
			"recipient":  req.Recipient,
		},
	})
}

// NotifyReservationCreated notifica cuando se crea una reserva
func (h *NotificationHandler) NotifyReservationCreated(c *gin.Context) {
	var req model.ReservationNotification

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Datos de notificación de reserva inválidos",
			"details": err.Error(),
		})
		return
	}

	// Validar campos requeridos
	if req.ReservationID == "" || req.EventID == "" || req.EventName == "" || req.Recipient == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ReservationID, EventID, EventName y Recipient son campos requeridos",
		})
		return
	}

	// Enviar notificación
	if err := h.notificationService.NotifyReservationCreated(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error enviando notificación de reserva",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notificación de reserva enviada exitosamente",
		"data": gin.H{
			"reservation_id": req.ReservationID,
			"event_id":       req.EventID,
			"event_name":     req.EventName,
			"recipient":      req.Recipient,
		},
	})
}

// NotifyReservationConfirmed notifica cuando se confirma una reserva
func (h *NotificationHandler) NotifyReservationConfirmed(c *gin.Context) {
	var req model.ReservationNotification

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Datos de confirmación de reserva inválidos",
			"details": err.Error(),
		})
		return
	}

	// Validar campos requeridos
	if req.ReservationID == "" || req.EventID == "" || req.EventName == "" || req.Recipient == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ReservationID, EventID, EventName y Recipient son campos requeridos",
		})
		return
	}

	// Enviar notificación
	if err := h.notificationService.NotifyReservationConfirmed(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error enviando notificación de confirmación de reserva",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notificación de confirmación de reserva enviada exitosamente",
		"data": gin.H{
			"reservation_id": req.ReservationID,
			"event_id":       req.EventID,
			"event_name":     req.EventName,
			"recipient":      req.Recipient,
		},
	})
}

// NotifyReservationCancelled notifica cuando se cancela una reserva
func (h *NotificationHandler) NotifyReservationCancelled(c *gin.Context) {
	var req model.ReservationNotification

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Datos de cancelación de reserva inválidos",
			"details": err.Error(),
		})
		return
	}

	// Validar campos requeridos
	if req.ReservationID == "" || req.EventID == "" || req.EventName == "" || req.Recipient == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ReservationID, EventID, EventName y Recipient son campos requeridos",
		})
		return
	}

	// Enviar notificación
	if err := h.notificationService.NotifyReservationCancelled(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error enviando notificación de cancelación de reserva",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notificación de cancelación de reserva enviada exitosamente",
		"data": gin.H{
			"reservation_id": req.ReservationID,
			"event_id":       req.EventID,
			"event_name":     req.EventName,
			"recipient":      req.Recipient,
		},
	})
}

