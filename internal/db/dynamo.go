package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/jhonathanssegura/ticket-notification/internal/model"
)

type DynamoClient struct {
	Client *dynamodb.Client
}

// SaveNotification guarda una notificación en DynamoDB
func (d *DynamoClient) SaveNotification(notification model.Notification) error {
	fmt.Printf("Guardando notificación: ID=%s, Type=%s, Recipient=%s\n",
		notification.ID.String(), notification.Type, notification.Recipient)

	item := map[string]types.AttributeValue{
		"id":          &types.AttributeValueMemberS{Value: notification.ID.String()},
		"type":        &types.AttributeValueMemberS{Value: string(notification.Type)},
		"status":      &types.AttributeValueMemberS{Value: string(notification.Status)},
		"priority":    &types.AttributeValueMemberS{Value: string(notification.Priority)},
		"recipient":   &types.AttributeValueMemberS{Value: notification.Recipient},
		"subject":     &types.AttributeValueMemberS{Value: notification.Subject},
		"content":     &types.AttributeValueMemberS{Value: notification.Content},
		"template_id": &types.AttributeValueMemberS{Value: notification.TemplateID},
		"created_at":  &types.AttributeValueMemberS{Value: notification.CreatedAt.Format(time.RFC3339)},
		"updated_at":  &types.AttributeValueMemberS{Value: notification.UpdatedAt.Format(time.RFC3339)},
	}

	// Campos opcionales
	if notification.SentAt != nil {
		item["sent_at"] = &types.AttributeValueMemberS{Value: notification.SentAt.Format(time.RFC3339)}
	}
	if notification.ReadAt != nil {
		item["read_at"] = &types.AttributeValueMemberS{Value: notification.ReadAt.Format(time.RFC3339)}
	}

	// Convertir datos adicionales a JSON string (simplificado)
	if len(notification.Data) > 0 {
		dataStr := fmt.Sprintf("%v", notification.Data)
		item["data"] = &types.AttributeValueMemberS{Value: dataStr}
	}

	_, err := d.Client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String("notifications"),
		Item:      item,
	})

	if err != nil {
		var errorMsg string
		switch {
		case strings.Contains(err.Error(), "ResourceNotFoundException"):
			errorMsg = "La tabla 'notifications' no existe en DynamoDB. Verifique que LocalStack esté ejecutándose y la tabla haya sido creada."
		case strings.Contains(err.Error(), "RequestCanceled"):
			errorMsg = "Error de conexión con DynamoDB. Verifique que LocalStack esté ejecutándose en http://localhost:4566."
		case strings.Contains(err.Error(), "ConditionalCheckFailedException"):
			errorMsg = "La notificación ya existe en la base de datos."
		default:
			errorMsg = fmt.Sprintf("Error guardando notificación en DynamoDB: %v", err)
		}
		return fmt.Errorf(errorMsg)
	}

	return nil
}

// GetNotificationByID obtiene una notificación por ID
func (d *DynamoClient) GetNotificationByID(notificationID string) (*model.Notification, error) {
	result, err := d.Client.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String("notifications"),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: notificationID},
		},
	})
	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, errors.New("notification not found")
	}

	notification, err := d.unmarshalNotification(result.Item)
	if err != nil {
		return nil, err
	}

	return notification, nil
}

// GetNotifications obtiene notificaciones con filtros opcionales
func (d *DynamoClient) GetNotifications(recipient string, notificationType string, limit int) ([]model.Notification, error) {
	scanInput := &dynamodb.ScanInput{
		TableName: aws.String("notifications"),
		Limit:     aws.Int32(int32(limit)),
	}

	// Aplicar filtros si se especifican
	if recipient != "" || notificationType != "" {
		var filterExpressions []string
		var expressionAttributeNames map[string]string
		var expressionAttributeValues map[string]types.AttributeValue

		if recipient != "" {
			filterExpressions = append(filterExpressions, "#recipient = :recipient")
			if expressionAttributeNames == nil {
				expressionAttributeNames = make(map[string]string)
			}
			expressionAttributeNames["#recipient"] = "recipient"
			if expressionAttributeValues == nil {
				expressionAttributeValues = make(map[string]types.AttributeValue)
			}
			expressionAttributeValues[":recipient"] = &types.AttributeValueMemberS{Value: recipient}
		}

		if notificationType != "" {
			filterExpressions = append(filterExpressions, "#type = :type")
			if expressionAttributeNames == nil {
				expressionAttributeNames = make(map[string]string)
			}
			expressionAttributeNames["#type"] = "type"
			if expressionAttributeValues == nil {
				expressionAttributeValues = make(map[string]types.AttributeValue)
			}
			expressionAttributeValues[":type"] = &types.AttributeValueMemberS{Value: notificationType}
		}

		if len(filterExpressions) > 0 {
			scanInput.FilterExpression = aws.String(strings.Join(filterExpressions, " AND "))
			scanInput.ExpressionAttributeNames = expressionAttributeNames
			scanInput.ExpressionAttributeValues = expressionAttributeValues
		}
	}

	result, err := d.Client.Scan(context.TODO(), scanInput)
	if err != nil {
		return nil, err
	}

	var notifications []model.Notification
	for _, item := range result.Items {
		notification, err := d.unmarshalNotification(item)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, *notification)
	}

	return notifications, nil
}

// UpdateNotification actualiza una notificación existente
func (d *DynamoClient) UpdateNotification(notificationID string, updates map[string]interface{}) error {
	var updateExpressions []string
	var expressionAttributeNames map[string]string
	var expressionAttributeValues map[string]types.AttributeValue

	for key, value := range updates {
		if updateExpressions == nil {
			updateExpressions = make([]string, 0)
			expressionAttributeNames = make(map[string]string)
			expressionAttributeValues = make(map[string]types.AttributeValue)
		}

		attrName := fmt.Sprintf("#%s", key)
		attrValue := fmt.Sprintf(":%s", key)

		updateExpressions = append(updateExpressions, fmt.Sprintf("%s = %s", attrName, attrValue))
		expressionAttributeNames[attrName] = key

		switch v := value.(type) {
		case string:
			expressionAttributeValues[attrValue] = &types.AttributeValueMemberS{Value: v}
		case time.Time:
			expressionAttributeValues[attrValue] = &types.AttributeValueMemberS{Value: v.Format(time.RFC3339)}
		case *time.Time:
			if v != nil {
				expressionAttributeValues[attrValue] = &types.AttributeValueMemberS{Value: v.Format(time.RFC3339)}
			}
		default:
			expressionAttributeValues[attrValue] = &types.AttributeValueMemberS{Value: fmt.Sprintf("%v", v)}
		}
	}

	// Agregar updated_at
	updateExpressions = append(updateExpressions, "#updated_at = :updated_at")
	expressionAttributeNames["#updated_at"] = "updated_at"
	expressionAttributeValues[":updated_at"] = &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)}

	_, err := d.Client.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
		TableName: aws.String("notifications"),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: notificationID},
		},
		UpdateExpression:          aws.String("SET " + strings.Join(updateExpressions, ", ")),
		ExpressionAttributeNames:  expressionAttributeNames,
		ExpressionAttributeValues: expressionAttributeValues,
	})

	return err
}

// DeleteNotification elimina una notificación
func (d *DynamoClient) DeleteNotification(notificationID string) error {
	_, err := d.Client.DeleteItem(context.TODO(), &dynamodb.DeleteItemInput{
		TableName: aws.String("notifications"),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: notificationID},
		},
	})
	return err
}

// SaveNotificationTemplate guarda una plantilla de notificación
func (d *DynamoClient) SaveNotificationTemplate(template model.NotificationTemplate) error {
	fmt.Printf("Guardando plantilla: ID=%s, Name=%s, Type=%s\n",
		template.ID.String(), template.Name, template.Type)

	item := map[string]types.AttributeValue{
		"id":         &types.AttributeValueMemberS{Value: template.ID.String()},
		"name":       &types.AttributeValueMemberS{Value: template.Name},
		"type":       &types.AttributeValueMemberS{Value: string(template.Type)},
		"subject":    &types.AttributeValueMemberS{Value: template.Subject},
		"content":    &types.AttributeValueMemberS{Value: template.Content},
		"is_active":  &types.AttributeValueMemberBOOL{Value: template.IsActive},
		"created_at": &types.AttributeValueMemberS{Value: template.CreatedAt.Format(time.RFC3339)},
		"updated_at": &types.AttributeValueMemberS{Value: template.UpdatedAt.Format(time.RFC3339)},
	}

	// Convertir variables a string (simplificado)
	if len(template.Variables) > 0 {
		variablesStr := strings.Join(template.Variables, ",")
		item["variables"] = &types.AttributeValueMemberS{Value: variablesStr}
	}

	_, err := d.Client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String("notification_templates"),
		Item:      item,
	})

	if err != nil {
		var errorMsg string
		switch {
		case strings.Contains(err.Error(), "ResourceNotFoundException"):
			errorMsg = "La tabla 'notification_templates' no existe en DynamoDB. Verifique que LocalStack esté ejecutándose y la tabla haya sido creada."
		case strings.Contains(err.Error(), "RequestCanceled"):
			errorMsg = "Error de conexión con DynamoDB. Verifique que LocalStack esté ejecutándose en http://localhost:4566."
		default:
			errorMsg = fmt.Sprintf("Error guardando plantilla en DynamoDB: %v", err)
		}
		return fmt.Errorf(errorMsg)
	}

	return nil
}

// GetNotificationTemplate obtiene una plantilla por ID
func (d *DynamoClient) GetNotificationTemplate(templateID string) (*model.NotificationTemplate, error) {
	result, err := d.Client.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String("notification_templates"),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: templateID},
		},
	})
	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, errors.New("template not found")
	}

	template, err := d.unmarshalNotificationTemplate(result.Item)
	if err != nil {
		return nil, err
	}

	return template, nil
}

// unmarshalNotification convierte un item de DynamoDB a Notification
func (d *DynamoClient) unmarshalNotification(item map[string]types.AttributeValue) (*model.Notification, error) {
	notification := &model.Notification{}

	if idVal, ok := item["id"].(*types.AttributeValueMemberS); ok {
		id, err := uuid.Parse(idVal.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid notification ID: %v", err)
		}
		notification.ID = id
	}

	if typeVal, ok := item["type"].(*types.AttributeValueMemberS); ok {
		notification.Type = model.NotificationType(typeVal.Value)
	}

	if statusVal, ok := item["status"].(*types.AttributeValueMemberS); ok {
		notification.Status = model.NotificationStatus(statusVal.Value)
	}

	if priorityVal, ok := item["priority"].(*types.AttributeValueMemberS); ok {
		notification.Priority = model.NotificationPriority(priorityVal.Value)
	}

	if recipientVal, ok := item["recipient"].(*types.AttributeValueMemberS); ok {
		notification.Recipient = recipientVal.Value
	}

	if subjectVal, ok := item["subject"].(*types.AttributeValueMemberS); ok {
		notification.Subject = subjectVal.Value
	}

	if contentVal, ok := item["content"].(*types.AttributeValueMemberS); ok {
		notification.Content = contentVal.Value
	}

	if templateIDVal, ok := item["template_id"].(*types.AttributeValueMemberS); ok {
		notification.TemplateID = templateIDVal.Value
	}

	if createdAtVal, ok := item["created_at"].(*types.AttributeValueMemberS); ok {
		createdAt, err := time.Parse(time.RFC3339, createdAtVal.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid created_at time: %v", err)
		}
		notification.CreatedAt = createdAt
	}

	if updatedAtVal, ok := item["updated_at"].(*types.AttributeValueMemberS); ok {
		updatedAt, err := time.Parse(time.RFC3339, updatedAtVal.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid updated_at time: %v", err)
		}
		notification.UpdatedAt = updatedAt
	}

	// Campos opcionales
	if sentAtVal, ok := item["sent_at"].(*types.AttributeValueMemberS); ok {
		sentAt, err := time.Parse(time.RFC3339, sentAtVal.Value)
		if err == nil {
			notification.SentAt = &sentAt
		}
	}

	if readAtVal, ok := item["read_at"].(*types.AttributeValueMemberS); ok {
		readAt, err := time.Parse(time.RFC3339, readAtVal.Value)
		if err == nil {
			notification.ReadAt = &readAt
		}
	}

	return notification, nil
}

// unmarshalNotificationTemplate convierte un item de DynamoDB a NotificationTemplate
func (d *DynamoClient) unmarshalNotificationTemplate(item map[string]types.AttributeValue) (*model.NotificationTemplate, error) {
	template := &model.NotificationTemplate{}

	if idVal, ok := item["id"].(*types.AttributeValueMemberS); ok {
		id, err := uuid.Parse(idVal.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid template ID: %v", err)
		}
		template.ID = id
	}

	if nameVal, ok := item["name"].(*types.AttributeValueMemberS); ok {
		template.Name = nameVal.Value
	}

	if typeVal, ok := item["type"].(*types.AttributeValueMemberS); ok {
		template.Type = model.NotificationType(typeVal.Value)
	}

	if subjectVal, ok := item["subject"].(*types.AttributeValueMemberS); ok {
		template.Subject = subjectVal.Value
	}

	if contentVal, ok := item["content"].(*types.AttributeValueMemberS); ok {
		template.Content = contentVal.Value
	}

	if isActiveVal, ok := item["is_active"].(*types.AttributeValueMemberBOOL); ok {
		template.IsActive = isActiveVal.Value
	}

	if createdAtVal, ok := item["created_at"].(*types.AttributeValueMemberS); ok {
		createdAt, err := time.Parse(time.RFC3339, createdAtVal.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid created_at time: %v", err)
		}
		template.CreatedAt = createdAt
	}

	if updatedAtVal, ok := item["updated_at"].(*types.AttributeValueMemberS); ok {
		updatedAt, err := time.Parse(time.RFC3339, updatedAtVal.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid updated_at time: %v", err)
		}
		template.UpdatedAt = updatedAt
	}

	// Variables (simplificado)
	if variablesVal, ok := item["variables"].(*types.AttributeValueMemberS); ok {
		template.Variables = strings.Split(variablesVal.Value, ",")
	}

	return template, nil
}

