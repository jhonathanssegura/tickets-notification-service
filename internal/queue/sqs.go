package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// NotificationMessage representa un mensaje de notificación en la cola SQS
type NotificationMessage struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Priority   string                 `json:"priority"`
	Recipient  string                 `json:"recipient"`
	Subject    string                 `json:"subject"`
	Content    string                 `json:"content"`
	TemplateID string                 `json:"template_id"`
	Data       map[string]interface{} `json:"data"`
	RetryCount int                    `json:"retry_count"`
	CreatedAt  string                 `json:"created_at"`
}

// EventNotificationMessage representa un mensaje de notificación de evento
type EventNotificationMessage struct {
	EventID    string `json:"event_id"`
	EventName  string `json:"event_name"`
	EventDate  string `json:"event_date"`
	Location   string `json:"location"`
	Recipient  string `json:"recipient"`
	Type       string `json:"type"`
	Priority   string `json:"priority"`
	TemplateID string `json:"template_id"`
}

// ReservationNotificationMessage representa un mensaje de notificación de reserva
type ReservationNotificationMessage struct {
	ReservationID string `json:"reservation_id"`
	EventID       string `json:"event_id"`
	EventName     string `json:"event_name"`
	EventDate     string `json:"event_date"`
	Location      string `json:"location"`
	Recipient     string `json:"recipient"`
	Type          string `json:"type"`
	Priority      string `json:"priority"`
	TemplateID    string `json:"template_id"`
}

// ReminderMessage representa un mensaje de recordatorio
type ReminderMessage struct {
	EventID      string `json:"event_id"`
	EventName    string `json:"event_name"`
	EventDate    string `json:"event_date"`
	Location     string `json:"location"`
	Recipient    string `json:"recipient"`
	ReminderType string `json:"reminder_type"` // "24h_before", "1h_before", "15min_before"
	TemplateID   string `json:"template_id"`
}

// SQSClient maneja las operaciones con las colas SQS
type SQSClient struct {
	Client   *sqs.Client
	QueueURL string
}

// SendMessage envía un mensaje simple a la cola
func (s *SQSClient) SendMessage(message string) error {
	ctx := context.Background()
	_, err := s.Client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.QueueURL),
		MessageBody: aws.String(message),
	})
	if err != nil {
		return fmt.Errorf("error sending SQS message: %w", err)
	}
	return nil
}

// SendNotificationMessage envía un mensaje de notificación estructurado
func (s *SQSClient) SendNotificationMessage(ctx context.Context, msg NotificationMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("error marshaling notification message: %w", err)
	}

	_, err = s.Client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.QueueURL),
		MessageBody: aws.String(string(body)),
		MessageAttributes: map[string]sqs.MessageAttributeValue{
			"Type": {
				DataType:    aws.String("String"),
				StringValue: aws.String(msg.Type),
			},
			"Priority": {
				DataType:    aws.String("String"),
				StringValue: aws.String(msg.Priority),
			},
			"Recipient": {
				DataType:    aws.String("String"),
				StringValue: aws.String(msg.Recipient),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("error sending notification message: %w", err)
	}
	return nil
}

// SendEventNotification envía una notificación de evento
func (s *SQSClient) SendEventNotification(ctx context.Context, msg EventNotificationMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("error marshaling event notification message: %w", err)
	}

	_, err = s.Client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.QueueURL),
		MessageBody: aws.String(string(body)),
		MessageAttributes: map[string]sqs.MessageAttributeValue{
			"Type": {
				DataType:    aws.String("String"),
				StringValue: aws.String("event_notification"),
			},
			"EventID": {
				DataType:    aws.String("String"),
				StringValue: aws.String(msg.EventID),
			},
			"Priority": {
				DataType:    aws.String("String"),
				StringValue: aws.String(msg.Priority),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("error sending event notification: %w", err)
	}
	return nil
}

// SendReservationNotification envía una notificación de reserva
func (s *SQSClient) SendReservationNotification(ctx context.Context, msg ReservationNotificationMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("error marshaling reservation notification message: %w", err)
	}

	_, err = s.Client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.QueueURL),
		MessageBody: aws.String(string(body)),
		MessageAttributes: map[string]sqs.MessageAttributeValue{
			"Type": {
				DataType:    aws.String("String"),
				StringValue: aws.String("reservation_notification"),
			},
			"ReservationID": {
				DataType:    aws.String("String"),
				StringValue: aws.String(msg.ReservationID),
			},
			"Priority": {
				DataType:    aws.String("String"),
				StringValue: aws.String(msg.Priority),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("error sending reservation notification: %w", err)
	}
	return nil
}

// SendReminderMessage envía un mensaje de recordatorio
func (s *SQSClient) SendReminderMessage(ctx context.Context, msg ReminderMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("error marshaling reminder message: %w", err)
	}

	_, err = s.Client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.QueueURL),
		MessageBody: aws.String(string(body)),
		MessageAttributes: map[string]sqs.MessageAttributeValue{
			"Type": {
				DataType:    aws.String("String"),
				StringValue: aws.String("reminder"),
			},
			"EventID": {
				DataType:    aws.String("String"),
				StringValue: aws.String(msg.EventID),
			},
			"ReminderType": {
				DataType:    aws.String("String"),
				StringValue: aws.String(msg.ReminderType),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("error sending reminder message: %w", err)
	}
	return nil
}

// ReceiveMessages recibe mensajes de la cola
func (s *SQSClient) ReceiveMessages(ctx context.Context, maxMessages int32) ([]sqs.Message, error) {
	resp, err := s.Client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(s.QueueURL),
		MaxNumberOfMessages: maxMessages,
		WaitTimeSeconds:     10,
		MessageAttributeNames: []string{
			"All",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error receiving SQS messages: %w", err)
	}

	return resp.Messages, nil
}

// DeleteMessage elimina un mensaje de la cola
func (s *SQSClient) DeleteMessage(ctx context.Context, receiptHandle string) error {
	_, err := s.Client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(s.QueueURL),
		ReceiptHandle: aws.String(receiptHandle),
	})
	if err != nil {
		return fmt.Errorf("error deleting SQS message: %w", err)
	}
	return nil
}

// GetQueueAttributes obtiene atributos de la cola
func (s *SQSClient) GetQueueAttributes(ctx context.Context) (*sqs.GetQueueAttributesOutput, error) {
	resp, err := s.Client.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl: aws.String(s.QueueURL),
		AttributeNames: []sqs.QueueAttributeName{
			"ApproximateNumberOfMessages",
			"ApproximateNumberOfMessagesNotVisible",
			"ApproximateNumberOfMessagesDelayed",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error getting queue attributes: %w", err)
	}

	return resp, nil
}

// PurgeQueue purga todos los mensajes de la cola
func (s *SQSClient) PurgeQueue(ctx context.Context) error {
	_, err := s.Client.PurgeQueue(ctx, &sqs.PurgeQueueInput{
		QueueUrl: aws.String(s.QueueURL),
	})
	if err != nil {
		return fmt.Errorf("error purging queue: %w", err)
	}
	return nil
}

