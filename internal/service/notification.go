package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/google/uuid"
	"github.com/jhonathanssegura/ticket-notification/internal/model"
	"github.com/jhonathanssegura/ticket-notification/internal/queue"
)

// NotificationService maneja el envío y gestión de notificaciones
type NotificationService struct {
	sesClient        *ses.Client
	eventQueue       *queue.SQSClient
	reservationQueue *queue.SQSClient
	reminderQueue    *queue.SQSClient
}

// NewNotificationService crea una nueva instancia del servicio de notificaciones
func NewNotificationService(
	sesClient *ses.Client,
	eventQueue *queue.SQSClient,
	reservationQueue *queue.SQSClient,
	reminderQueue *queue.SQSClient,
) *NotificationService {
	return &NotificationService{
		sesClient:        sesClient,
		eventQueue:       eventQueue,
		reservationQueue: reservationQueue,
		reminderQueue:    reminderQueue,
	}
}

// SendNotification envía una notificación individual
func (s *NotificationService) SendNotification(ctx context.Context, req model.CreateNotificationRequest) (*model.Notification, error) {
	notification := &model.Notification{
		ID:         uuid.New(),
		Type:       req.Type,
		Status:     model.NotificationStatusPending,
		Priority:   req.Priority,
		Recipient:  req.Recipient,
		Subject:    req.Subject,
		Content:    req.Content,
		TemplateID: req.TemplateID,
		Data:       req.Data,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Si no se especifica prioridad, usar normal
	if notification.Priority == "" {
		notification.Priority = model.NotificationPriorityNormal
	}

	// Enviar por email
	if err := s.sendEmailNotification(ctx, notification); err != nil {
		log.Printf("Error enviando email: %v", err)
		notification.Status = model.NotificationStatusFailed
	} else {
		now := time.Now()
		notification.Status = model.NotificationStatusSent
		notification.SentAt = &now
	}

	return notification, nil
}

// SendBulkNotifications envía múltiples notificaciones
func (s *NotificationService) SendBulkNotifications(ctx context.Context, req model.BulkNotificationRequest) ([]*model.Notification, error) {
	var notifications []*model.Notification
	var errors []error

	for _, notificationReq := range req.Notifications {
		// Aplicar prioridad global si se especifica
		if req.Priority != "" {
			notificationReq.Priority = req.Priority
		}

		// Aplicar template global si se especifica
		if req.TemplateID != "" {
			notificationReq.TemplateID = req.TemplateID
		}

		notification, err := s.SendNotification(ctx, notificationReq)
		if err != nil {
			errors = append(errors, fmt.Errorf("error sending notification to %s: %w", notificationReq.Recipient, err))
		} else {
			notifications = append(notifications, notification)
		}
	}

	if len(errors) > 0 {
		log.Printf("Errors sending bulk notifications: %v", errors)
	}

	return notifications, nil
}

// NotifyEventCreated notifica cuando se crea un evento
func (s *NotificationService) NotifyEventCreated(ctx context.Context, req model.EventNotification) error {
	// Crear mensaje para la cola de eventos
	msg := queue.EventNotificationMessage{
		EventID:    req.EventID,
		EventName:  req.EventName,
		EventDate:  req.EventDate.Format(time.RFC3339),
		Location:   req.Location,
		Recipient:  req.Recipient,
		Type:       string(req.Type),
		Priority:   string(req.Priority),
		TemplateID: "event_created_template",
	}

	// Enviar a la cola de eventos
	if err := s.eventQueue.SendEventNotification(ctx, msg); err != nil {
		return fmt.Errorf("error sending event notification to queue: %w", err)
	}

	// También enviar email inmediato si es alta prioridad
	if req.Priority == model.NotificationPriorityHigh || req.Priority == model.NotificationPriorityUrgent {
		notification := &model.Notification{
			ID:        uuid.New(),
			Type:      req.Type,
			Status:    model.NotificationStatusPending,
			Priority:  req.Priority,
			Recipient: req.Recipient,
			Subject:   fmt.Sprintf("Nuevo Evento: %s", req.EventName),
			Content:   fmt.Sprintf("Se ha creado un nuevo evento: %s en %s el %s", req.EventName, req.Location, req.EventDate.Format("02/01/2006 15:04")),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := s.sendEmailNotification(ctx, notification); err != nil {
			log.Printf("Error sending immediate event notification email: %v", err)
		}
	}

	return nil
}

// SendEventReminder envía un recordatorio de evento
func (s *NotificationService) SendEventReminder(ctx context.Context, req model.EventNotification) error {
	// Crear mensaje para la cola de recordatorios
	msg := queue.ReminderMessage{
		EventID:      req.EventID,
		EventName:    req.EventName,
		EventDate:    req.EventDate.Format(time.RFC3339),
		Location:     req.Location,
		Recipient:    req.Recipient,
		ReminderType: "event_reminder",
		TemplateID:   "event_reminder_template",
	}

	// Enviar a la cola de recordatorios
	if err := s.reminderQueue.SendReminderMessage(ctx, msg); err != nil {
		return fmt.Errorf("error sending reminder to queue: %w", err)
	}

	return nil
}

// NotifyEventCancelled notifica cuando se cancela un evento
func (s *NotificationService) NotifyEventCancelled(ctx context.Context, req model.EventNotification) error {
	// Crear mensaje para la cola de eventos
	msg := queue.EventNotificationMessage{
		EventID:    req.EventID,
		EventName:  req.EventName,
		EventDate:  req.EventDate.Format(time.RFC3339),
		Location:   req.Location,
		Recipient:  req.Recipient,
		Type:       string(req.Type),
		Priority:   string(req.Priority),
		TemplateID: "event_cancelled_template",
	}

	// Enviar a la cola de eventos
	if err := s.eventQueue.SendEventNotification(ctx, msg); err != nil {
		return fmt.Errorf("error sending event cancellation to queue: %w", err)
	}

	// Enviar email inmediato para cancelaciones
	notification := &model.Notification{
		ID:        uuid.New(),
		Type:      req.Type,
		Status:    model.NotificationStatusPending,
		Priority:  req.Priority,
		Recipient: req.Recipient,
		Subject:   fmt.Sprintf("Evento Cancelado: %s", req.EventName),
		Content:   fmt.Sprintf("El evento '%s' programado para el %s en %s ha sido cancelado.", req.EventName, req.EventDate.Format("02/01/2006 15:04"), req.Location),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.sendEmailNotification(ctx, notification); err != nil {
		log.Printf("Error sending event cancellation email: %v", err)
	}

	return nil
}

// NotifyReservationCreated notifica cuando se crea una reserva
func (s *NotificationService) NotifyReservationCreated(ctx context.Context, req model.ReservationNotification) error {
	// Crear mensaje para la cola de reservas
	msg := queue.ReservationNotificationMessage{
		ReservationID: req.ReservationID,
		EventID:       req.EventID,
		EventName:     req.EventName,
		EventDate:     req.EventDate.Format(time.RFC3339),
		Location:      req.Location,
		Recipient:     req.Recipient,
		Type:          string(req.Type),
		Priority:      string(req.Priority),
		TemplateID:    "reservation_created_template",
	}

	// Enviar a la cola de reservas
	if err := s.reservationQueue.SendReservationNotification(ctx, msg); err != nil {
		return fmt.Errorf("error sending reservation notification to queue: %w", err)
	}

	// Enviar email de confirmación inmediata
	notification := &model.Notification{
		ID:        uuid.New(),
		Type:      req.Type,
		Status:    model.NotificationStatusPending,
		Priority:  req.Priority,
		Recipient: req.Recipient,
		Subject:   fmt.Sprintf("Reserva Confirmada: %s", req.EventName),
		Content:   fmt.Sprintf("Tu reserva para el evento '%s' el %s en %s ha sido confirmada. ID de reserva: %s", req.EventName, req.EventDate.Format("02/01/2006 15:04"), req.Location, req.ReservationID),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.sendEmailNotification(ctx, notification); err != nil {
		log.Printf("Error sending reservation confirmation email: %v", err)
	}

	return nil
}

// NotifyReservationConfirmed notifica cuando se confirma una reserva
func (s *NotificationService) NotifyReservationConfirmed(ctx context.Context, req model.ReservationNotification) error {
	// Crear mensaje para la cola de reservas
	msg := queue.ReservationNotificationMessage{
		ReservationID: req.ReservationID,
		EventID:       req.EventID,
		EventName:     req.EventName,
		EventDate:     req.EventDate.Format(time.RFC3339),
		Location:      req.Location,
		Recipient:     req.Recipient,
		Type:          string(req.Type),
		Priority:      string(req.Priority),
		TemplateID:    "reservation_confirmed_template",
	}

	// Enviar a la cola de reservas
	if err := s.reservationQueue.SendReservationNotification(ctx, msg); err != nil {
		return fmt.Errorf("error sending reservation confirmation to queue: %w", err)
	}

	return nil
}

// NotifyReservationCancelled notifica cuando se cancela una reserva
func (s *NotificationService) NotifyReservationCancelled(ctx context.Context, req model.ReservationNotification) error {
	// Crear mensaje para la cola de reservas
	msg := queue.ReservationNotificationMessage{
		ReservationID: req.ReservationID,
		EventID:       req.EventID,
		EventName:     req.EventName,
		EventDate:     req.EventDate.Format(time.RFC3339),
		Location:      req.Location,
		Recipient:     req.Recipient,
		Type:          string(req.Type),
		Priority:      string(req.Priority),
		TemplateID:    "reservation_cancelled_template",
	}

	// Enviar a la cola de reservas
	if err := s.reservationQueue.SendReservationNotification(ctx, msg); err != nil {
		return fmt.Errorf("error sending reservation cancellation to queue: %w", err)
	}

	// Enviar email inmediato para cancelaciones de reserva
	notification := &model.Notification{
		ID:        uuid.New(),
		Type:      req.Type,
		Status:    model.NotificationStatusPending,
		Priority:  req.Priority,
		Recipient: req.Recipient,
		Subject:   fmt.Sprintf("Reserva Cancelada: %s", req.EventName),
		Content:   fmt.Sprintf("Tu reserva para el evento '%s' el %s en %s ha sido cancelada. ID de reserva: %s", req.EventName, req.EventDate.Format("02/01/2006 15:04"), req.Location, req.ReservationID),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.sendEmailNotification(ctx, notification); err != nil {
		log.Printf("Error sending reservation cancellation email: %v", err)
	}

	return nil
}

// sendEmailNotification envía una notificación por email usando SES
func (s *NotificationService) sendEmailNotification(ctx context.Context, notification *model.Notification) error {
	// Configurar el email
	emailInput := &ses.SendEmailInput{
		Source: aws.String("notifications@ticket-system.com"),
		Destination: &ses.Destination{
			ToAddresses: []string{notification.Recipient},
		},
		Message: &ses.Message{
			Subject: &ses.Content{
				Data:    aws.String(notification.Subject),
				Charset: aws.String("UTF-8"),
			},
			Body: &ses.Body{
				Text: &ses.Content{
					Data:    aws.String(notification.Content),
					Charset: aws.String("UTF-8"),
				},
			},
		},
	}

	// Enviar el email
	_, err := s.sesClient.SendEmail(ctx, emailInput)
	if err != nil {
		return fmt.Errorf("error sending email via SES: %w", err)
	}

	log.Printf("Email notification sent successfully to %s", notification.Recipient)
	return nil
}

// ProcessNotificationQueue procesa la cola de notificaciones
func (s *NotificationService) ProcessNotificationQueue(ctx context.Context, queueType string) error {
	var client *queue.SQSClient

	switch queueType {
	case "events":
		client = s.eventQueue
	case "reservations":
		client = s.reservationQueue
	case "reminders":
		client = s.reminderQueue
	default:
		return fmt.Errorf("invalid queue type: %s", queueType)
	}

	// Recibir mensajes de la cola
	messages, err := client.ReceiveMessages(ctx, 10)
	if err != nil {
		return fmt.Errorf("error receiving messages: %w", err)
	}

	log.Printf("Processing %d messages from %s queue", len(messages), queueType)

	for _, message := range messages {
		// Procesar el mensaje según el tipo
		if err := s.processMessage(ctx, message, queueType); err != nil {
			log.Printf("Error processing message %s: %v", *message.MessageId, err)
			continue
		}

		// Eliminar el mensaje procesado
		if err := client.DeleteMessage(ctx, *message.ReceiptHandle); err != nil {
			log.Printf("Error deleting message %s: %v", *message.MessageId, err)
		}
	}

	return nil
}

// processMessage procesa un mensaje individual de la cola
func (s *NotificationService) processMessage(ctx context.Context, message sqs.Message, queueType string) error {
	log.Printf("Processing message %s from %s queue", *message.MessageId, queueType)

	// Aquí se implementaría la lógica específica para cada tipo de mensaje
	// Por ahora, solo logueamos el procesamiento
	switch queueType {
	case "events":
		log.Printf("Processing event notification: %s", *message.Body)
	case "reservations":
		log.Printf("Processing reservation notification: %s", *message.Body)
	case "reminders":
		log.Printf("Processing reminder: %s", *message.Body)
	}

	return nil
}

