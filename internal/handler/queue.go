package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jhonathanssegura/ticket-notification/internal/db"
	"github.com/jhonathanssegura/ticket-notification/internal/service"
)

// QueueHandler maneja las peticiones HTTP relacionadas con las colas de notificaciones
type QueueHandler struct {
	notificationService *service.NotificationService
	dbClient            *db.DynamoClient
}

// NewQueueHandler crea una nueva instancia del handler de colas
func NewQueueHandler(notificationService *service.NotificationService, dbClient *db.DynamoClient) *QueueHandler {
	return &QueueHandler{
		notificationService: notificationService,
		dbClient:            dbClient,
	}
}

// ProcessNotificationQueue procesa una cola de notificaciones específica
func (h *QueueHandler) ProcessNotificationQueue(c *gin.Context) {
	queueType := c.Query("type")
	if queueType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "El parámetro 'type' es requerido (events, reservations, reminders)",
		})
		return
	}

	// Validar tipo de cola
	validTypes := map[string]bool{
		"events":       true,
		"reservations": true,
		"reminders":    true,
	}

	if !validTypes[queueType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Tipo de cola inválido. Debe ser: events, reservations, o reminders",
		})
		return
	}

	// Procesar la cola
	if err := h.notificationService.ProcessNotificationQueue(c.Request.Context(), queueType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error procesando cola de notificaciones",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cola de notificaciones procesada exitosamente",
		"data": gin.H{
			"queue_type":   queueType,
			"processed_at": time.Now().Format(time.RFC3339),
		},
	})
}

// GetQueueStatus obtiene el estado de todas las colas
func (h *QueueHandler) GetQueueStatus(c *gin.Context) {
	// Obtener estado de la cola de eventos
	eventQueueStatus, err := h.notificationService.GetEventQueueStatus(c.Request.Context())
	if err != nil {
		log.Printf("Error obteniendo estado de cola de eventos: %v", err)
		eventQueueStatus = gin.H{"error": err.Error()}
	}

	// Obtener estado de la cola de reservas
	reservationQueueStatus, err := h.notificationService.GetReservationQueueStatus(c.Request.Context())
	if err != nil {
		log.Printf("Error obteniendo estado de cola de reservas: %v", err)
		reservationQueueStatus = gin.H{"error": err.Error()}
	}

	// Obtener estado de la cola de recordatorios
	reminderQueueStatus, err := h.notificationService.GetReminderQueueStatus(c.Request.Context())
	if err != nil {
		log.Printf("Error obteniendo estado de cola de recordatorios: %v", err)
		reminderQueueStatus = gin.H{"error": err.Error()}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"event_queue":       eventQueueStatus,
			"reservation_queue": reservationQueueStatus,
			"reminder_queue":    reminderQueueStatus,
			"timestamp":         time.Now().Format(time.RFC3339),
		},
	})
}

// PurgeQueue purga una cola específica
func (h *QueueHandler) PurgeQueue(c *gin.Context) {
	queueType := c.Query("type")
	if queueType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "El parámetro 'type' es requerido (events, reservations, reminders)",
		})
		return
	}

	// Validar tipo de cola
	validTypes := map[string]bool{
		"events":       true,
		"reservations": true,
		"reminders":    true,
	}

	if !validTypes[queueType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Tipo de cola inválido. Debe ser: events, reservations, o reminders",
		})
		return
	}

	// Confirmar acción
	confirm := c.Query("confirm")
	if confirm != "true" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Para purgar una cola, debe confirmar la acción agregando ?confirm=true",
			"warning": "Esta acción eliminará TODOS los mensajes de la cola de forma permanente",
		})
		return
	}

	// Purgar la cola
	if err := h.notificationService.PurgeQueue(c.Request.Context(), queueType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error purgando cola",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cola purgada exitosamente",
		"data": gin.H{
			"queue_type": queueType,
			"purged_at":  time.Now().Format(time.RFC3339),
		},
	})
}

// GetQueueMetrics obtiene métricas detalladas de las colas
func (h *QueueHandler) GetQueueMetrics(c *gin.Context) {
	// Obtener métricas de la cola de eventos
	eventMetrics, err := h.notificationService.GetEventQueueMetrics(c.Request.Context())
	if err != nil {
		log.Printf("Error obteniendo métricas de cola de eventos: %v", err)
		eventMetrics = gin.H{"error": err.Error()}
	}

	// Obtener métricas de la cola de reservas
	reservationMetrics, err := h.notificationService.GetReservationQueueMetrics(c.Request.Context())
	if err != nil {
		log.Printf("Error obteniendo métricas de cola de reservas: %v", err)
		reservationMetrics = gin.H{"error": err.Error()}
	}

	// Obtener métricas de la cola de recordatorios
	reminderMetrics, err := h.notificationService.GetReminderQueueMetrics(c.Request.Context())
	if err != nil {
		log.Printf("Error obteniendo métricas de cola de recordatorios: %v", err)
		reminderMetrics = gin.H{"error": err.Error()}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"event_queue_metrics":       eventMetrics,
			"reservation_queue_metrics": reservationMetrics,
			"reminder_queue_metrics":    reminderMetrics,
			"timestamp":                 time.Now().Format(time.RFC3339),
		},
	})
}

// RetryFailedNotifications reintenta notificaciones fallidas
func (h *QueueHandler) RetryFailedNotifications(c *gin.Context) {
	queueType := c.Query("type")
	if queueType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "El parámetro 'type' es requerido (events, reservations, reminders)",
		})
		return
	}

	// Validar tipo de cola
	validTypes := map[string]bool{
		"events":       true,
		"reservations": true,
		"reminders":    true,
	}

	if !validTypes[queueType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Tipo de cola inválido. Debe ser: events, reservations, o reminders",
		})
		return
	}

	// Reintentar notificaciones fallidas
	retryCount, err := h.notificationService.RetryFailedNotifications(c.Request.Context(), queueType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error reintentando notificaciones fallidas",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notificaciones fallidas reintentadas exitosamente",
		"data": gin.H{
			"queue_type":  queueType,
			"retry_count": retryCount,
			"retried_at":  time.Now().Format(time.RFC3339),
		},
	})
}

