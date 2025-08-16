#!/bin/bash

# Script para configurar recursos AWS en LocalStack para el servicio de notificaciones
# Ejecutar después de iniciar LocalStack

set -e

echo "🚀 Configurando recursos AWS para Tickets Notification Service..."

# Configurar variables de entorno para AWS CLI
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-east-1
export AWS_ENDPOINT_URL=http://localhost:4566

# Función para esperar a que LocalStack esté listo
wait_for_localstack() {
    echo "⏳ Esperando a que LocalStack esté listo..."
    until aws --endpoint-url=http://localhost:4566 sts get-caller-identity >/dev/null 2>&1; do
        echo "LocalStack no está listo, esperando..."
        sleep 2
    done
    echo "✅ LocalStack está listo!"
}

# Función para crear tabla DynamoDB
create_dynamodb_table() {
    local table_name=$1
    local partition_key=$2
    
    echo "📊 Creando tabla DynamoDB: $table_name"
    
    aws --endpoint-url=http://localhost:4566 dynamodb create-table \
        --table-name "$table_name" \
        --attribute-definitions AttributeName="$partition_key",AttributeType=S \
        --key-schema AttributeName="$partition_key",KeyType=HASH \
        --billing-mode PAY_PER_REQUEST \
        --region us-east-1
    
    echo "✅ Tabla $table_name creada exitosamente"
}

# Función para crear cola SQS
create_sqs_queue() {
    local queue_name=$1
    
    echo "📱 Creando cola SQS: $queue_name"
    
    aws --endpoint-url=http://localhost:4566 sqs create-queue \
        --queue-name "$queue_name" \
        --region us-east-1
    
    echo "✅ Cola $queue_name creada exitosamente"
}

# Función para verificar si un recurso existe
resource_exists() {
    local resource_type=$1
    local resource_name=$2
    
    case $resource_type in
        "dynamodb")
            aws --endpoint-url=http://localhost:4566 dynamodb describe-table \
                --table-name "$resource_name" >/dev/null 2>&1
            ;;
        "sqs")
            aws --endpoint-url=http://localhost:4566 sqs get-queue-url \
                --queue-name "$resource_name" >/dev/null 2>&1
            ;;
        *)
            return 1
            ;;
    esac
}

# Esperar a que LocalStack esté listo
wait_for_localstack

echo "🔧 Creando recursos AWS..."

# Crear tablas DynamoDB
echo "📊 Configurando DynamoDB..."

if ! resource_exists "dynamodb" "notifications"; then
    create_dynamodb_table "notifications" "id"
else
    echo "ℹ️  Tabla 'notifications' ya existe"
fi

if ! resource_exists "dynamodb" "notification_templates"; then
    create_dynamodb_table "notification_templates" "id"
else
    echo "ℹ️  Tabla 'notification_templates' ya existe"
fi

# Crear colas SQS
echo "📱 Configurando SQS..."

if ! resource_exists "sqs" "event-notifications"; then
    create_sqs_queue "event-notifications"
else
    echo "ℹ️  Cola 'event-notifications' ya existe"
fi

if ! resource_exists "sqs" "reservation-notifications"; then
    create_sqs_queue "reservation-notifications"
else
    echo "ℹ️  Cola 'reservation-notifications' ya existe"
fi

if ! resource_exists "sqs" "reminder-notifications"; then
    create_sqs_queue "reminder-notifications"
else
    echo "ℹ️  Cola 'reminder-notifications' ya existe"
fi

# Configurar SES (simulado en LocalStack)
echo "📧 Configurando SES..."
echo "ℹ️  SES se configura automáticamente en LocalStack"

# Verificar recursos creados
echo "🔍 Verificando recursos creados..."

echo "📊 Tablas DynamoDB:"
aws --endpoint-url=http://localhost:4566 dynamodb list-tables --region us-east-1

echo "📱 Colas SQS:"
aws --endpoint-url=http://localhost:4566 sqs list-queues --region us-east-1

echo ""
echo "🎉 Configuración completada exitosamente!"
echo ""
echo "📋 Resumen de recursos creados:"
echo "   • Tabla DynamoDB: notifications"
echo "   • Tabla DynamoDB: notification_templates"
echo "   • Cola SQS: event-notifications"
echo "   • Cola SQS: reservation-notifications"
echo "   • Cola SQS: reminder-notifications"
echo ""
echo "🚀 El servicio de notificaciones está listo para usar!"
echo "   Puerto: 8085"
echo "   Health Check: http://localhost:8085/health"
echo ""
echo "💡 Para probar el servicio:"
echo "   curl http://localhost:8085/health"
echo ""
echo "💡 Para ver el estado de las colas:"
echo "   curl http://localhost:8085/api/v1/queue/status"

