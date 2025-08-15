package main

import (
	"log"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gin-gonic/gin"
	"github.com/jhonathanssegura/ticket-notification/internal/awsconfig"
	"github.com/jhonathanssegura/ticket-notification/internal/db"
	"github.com/jhonathanssegura/ticket-notification/internal/handler"
	"github.com/jhonathanssegura/ticket-notification/internal/queue"
	"github.com/jhonathanssegura/ticket-notification/internal/service"
)

func main() {
	cfg, err := awsconfig.LoadAWSConfig()
	if err != nil {
		log.Fatalf("Error cargando configuración AWS: %v", err)
	}

	// Configuración de colas SQS para diferentes tipos de notificaciones
	eventQueueURL := "http://localhost:4566/000000000000/event-notifications"
	reservationQueueURL := "http://localhost:4566/000000000000/reservation-notifications"
	reminderQueueURL := "http://localhost:4566/000000000000/reminder-notifications"

	// Crear clientes AWS
	sqsClient := sqs.NewFromConfig(cfg)
	sesClient := ses.NewFromConfig(cfg)
	dynamoClient := dynamodb.NewFromConfig(cfg)

	// Crear clientes de cola
	eventQueue := &queue.SQSClient{
		Client:   sqsClient,
		QueueURL: eventQueueURL,
	}
	reservationQueue := &queue.SQSClient{
		Client:   sqsClient,
		QueueURL: reservationQueueURL,
	}
	reminderQueue := &queue.SQSClient{
		Client:   sqsClient,
		QueueURL: reminderQueueURL,
	}

	// Crear cliente de base de datos
	dbClient := &db.DynamoClient{
		Client: dynamoClient,
	}

	// Crear servicio de notificaciones
	notificationService := service.NewNotificationService(sesClient, eventQueue, reservationQueue, reminderQueue)

	// Crear handlers
	notificationHandler := handler.NewNotificationHandler(notificationService, dbClient)
	queueHandler := handler.NewQueueHandler(notificationService, dbClient)

	// Configurar rutas
	r := gin.Default()

	// Middleware de CORS
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "tickets-notification-service",
			"version": "1.0.0",
		})
	})

	// API routes
	api := r.Group("/api/v1")
	{
		// Notification endpoints
		api.POST("/notifications/send", notificationHandler.SendNotification)
		api.POST("/notifications/bulk", notificationHandler.SendBulkNotifications)
		api.GET("/notifications/:id", notificationHandler.GetNotification)
		api.GET("/notifications", notificationHandler.ListNotifications)
		api.PUT("/notifications/:id", notificationHandler.UpdateNotification)
		api.DELETE("/notifications/:id", notificationHandler.DeleteNotification)

		// Event notification endpoints
		api.POST("/notifications/events", notificationHandler.NotifyEventCreated)
		api.POST("/notifications/events/:id/reminder", notificationHandler.SendEventReminder)
		api.POST("/notifications/events/:id/cancelled", notificationHandler.NotifyEventCancelled)

		// Reservation notification endpoints
		api.POST("/notifications/reservations", notificationHandler.NotifyReservationCreated)
		api.POST("/notifications/reservations/:id/confirmed", notificationHandler.NotifyReservationConfirmed)
		api.POST("/notifications/reservations/:id/cancelled", notificationHandler.NotifyReservationCancelled)

		// Queue processing endpoints
		api.POST("/queue/process", queueHandler.ProcessNotificationQueue)
		api.GET("/queue/status", queueHandler.GetQueueStatus)
	}

	log.Println("🚀 Iniciando servicio de notificaciones en puerto 8085...")
	log.Println("📧 Servicio de notificaciones por email configurado")
	log.Println("📱 Colas SQS configuradas para eventos, reservas y recordatorios")

	if err := r.Run(":8085"); err != nil {
		log.Fatalf("Error iniciando servidor: %v", err)
	}
}

