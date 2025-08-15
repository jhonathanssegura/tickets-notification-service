# Tickets Notification Service

Servicio de notificaciones para el sistema de reserva de tickets, encargado de manejar todas las notificaciones relacionadas con eventos, reservas y recordatorios.

## 🚀 Características

- **Notificaciones por Email**: Envío de notificaciones usando AWS SES
- **Colas SQS**: Manejo asíncrono de notificaciones con diferentes prioridades
- **Múltiples Tipos de Notificaciones**:
  - Eventos creados, actualizados y cancelados
  - Reservas creadas, confirmadas y canceladas
  - Recordatorios de eventos
  - Notificaciones personalizadas
- **Base de Datos DynamoDB**: Almacenamiento de notificaciones y plantillas
- **API REST**: Endpoints para gestión y envío de notificaciones
- **Procesamiento de Colas**: Sistema de procesamiento automático de mensajes

## 🏗️ Arquitectura

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   API Gateway  │───▶│ Notification     │───▶│   AWS SES      │
│   (Port 8085)  │    │   Service        │    │   (Email)      │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌──────────────────┐
                       │   SQS Queues     │
                       │                  │
                       │ • Events         │
                       │ • Reservations   │
                       │ • Reminders      │
                       └──────────────────┘
                                │
                                ▼
                       ┌──────────────────┐
                       │   DynamoDB       │
                       │                  │
                       │ • Notifications  │
                       │ • Templates      │
                       └──────────────────┘
```

## 📋 Prerrequisitos

- Go 1.24.5 o superior
- Docker y Docker Compose
- LocalStack (para desarrollo local)

## 🛠️ Instalación

### 1. Clonar el repositorio
```bash
git clone <repository-url>
cd tickets-notification-service
```

### 2. Instalar dependencias
```bash
go mod download
```

### 3. Configurar variables de entorno
```bash
cp .env.example .env
# Editar .env con tus configuraciones
```

### 4. Iniciar servicios con Docker
```bash
docker-compose up -d
```

### 5. Verificar que el servicio esté corriendo
```bash
curl http://localhost:8085/health
```

## 🚀 Uso

### Endpoints Principales

#### Notificaciones
- `POST /api/v1/notifications/send` - Enviar notificación individual
- `POST /api/v1/notifications/bulk` - Enviar múltiples notificaciones
- `GET /api/v1/notifications/:id` - Obtener notificación por ID
- `GET /api/v1/notifications` - Listar notificaciones
- `PUT /api/v1/notifications/:id` - Actualizar notificación
- `DELETE /api/v1/notifications/:id` - Eliminar notificación

#### Notificaciones de Eventos
- `POST /api/v1/notifications/events` - Notificar evento creado
- `POST /api/v1/notifications/events/:id/reminder` - Enviar recordatorio
- `POST /api/v1/notifications/events/:id/cancelled` - Notificar evento cancelado

#### Notificaciones de Reservas
- `POST /api/v1/notifications/reservations` - Notificar reserva creada
- `POST /api/v1/notifications/reservations/:id/confirmed` - Notificar reserva confirmada
- `POST /api/v1/notifications/reservations/:id/cancelled` - Notificar reserva cancelada

#### Gestión de Colas
- `POST /api/v1/queue/process` - Procesar cola de notificaciones
- `GET /api/v1/queue/status` - Obtener estado de las colas

### Ejemplos de Uso

#### Enviar Notificación Individual
```bash
curl -X POST http://localhost:8085/api/v1/notifications/send \
  -H "Content-Type: application/json" \
  -d '{
    "type": "event_created",
    "priority": "high",
    "recipient": "user@example.com",
    "subject": "Nuevo Evento Disponible",
    "content": "Se ha creado un nuevo evento que te puede interesar."
  }'
```

#### Notificar Evento Creado
```bash
curl -X POST http://localhost:8085/api/v1/notifications/events \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "evt-001",
    "event_name": "Concierto de Rock",
    "event_date": "2024-12-25T20:00:00Z",
    "location": "Estadio Nacional",
    "recipient": "user@example.com",
    "type": "event_created",
    "priority": "normal"
  }'
```

#### Procesar Cola de Eventos
```bash
curl -X POST "http://localhost:8085/api/v1/queue/process?type=events"
```

## 🔧 Configuración

### Variables de Entorno

```bash
# AWS Configuration
AWS_ACCESS_KEY_ID=your_access_key
AWS_SECRET_ACCESS_KEY=your_secret_key
AWS_DEFAULT_REGION=us-east-1
AWS_ENDPOINT_URL=http://localhost:4566

# Service Configuration
SERVICE_PORT=8085
SERVICE_ENV=development

# Database Configuration
DYNAMODB_ENDPOINT=http://localhost:4566
DYNAMODB_REGION=us-east-1

# SQS Configuration
SQS_ENDPOINT=http://localhost:4566
SQS_REGION=us-east-1

# SES Configuration
SES_ENDPOINT=http://localhost:4566
SES_REGION=us-east-1
```

### Configuración de LocalStack

El servicio está configurado para usar LocalStack en desarrollo local, que emula los servicios AWS:

- **DynamoDB**: Puerto 4566
- **SQS**: Puerto 4566
- **SES**: Puerto 4566

## 📊 Monitoreo

### Health Check
```bash
curl http://localhost:8085/health
```

### Estado de las Colas
```bash
curl http://localhost:8085/api/v1/queue/status
```

### Métricas de las Colas
```bash
curl http://localhost:8085/api/v1/queue/metrics
```

## 🧪 Testing

### Ejecutar Tests
```bash
go test ./...
```

### Tests con Coverage
```bash
go test -cover ./...
```

### Tests de Integración
```bash
go test -tags=integration ./...
```

## 🚀 Despliegue

### Construir Imagen Docker
```bash
docker build -t tickets-notification-service .
```

### Ejecutar Contenedor
```bash
docker run -p 8085:8085 tickets-notification-service
```

### Con Docker Compose
```bash
docker-compose up -d
```

## 📝 Logs

### Ver Logs del Servicio
```bash
docker-compose logs -f notification-service
```

### Ver Logs de LocalStack
```bash
docker-compose logs -f localstack
```

## 🔍 Troubleshooting

### Problemas Comunes

1. **Error de conexión a LocalStack**
   - Verificar que LocalStack esté corriendo: `docker-compose ps`
   - Verificar logs: `docker-compose logs localstack`

2. **Error de permisos en DynamoDB**
   - Verificar que las tablas existan en LocalStack
   - Verificar configuración de AWS

3. **Error de envío de email**
   - Verificar configuración de SES
   - Verificar que el dominio esté verificado en SES

### Logs de Debug

Para habilitar logs detallados, agregar en el archivo de configuración:
```bash
DEBUG=true
LOG_LEVEL=debug
```

## 🤝 Contribución

1. Fork el proyecto
2. Crear una rama para tu feature (`git checkout -b feature/AmazingFeature`)
3. Commit tus cambios (`git commit -m 'Add some AmazingFeature'`)
4. Push a la rama (`git push origin feature/AmazingFeature`)
5. Abrir un Pull Request

## 📄 Licencia

Este proyecto está bajo la Licencia MIT. Ver el archivo `LICENSE` para más detalles.

## 📞 Soporte

Para soporte técnico o preguntas:
- Crear un issue en GitHub
- Contactar al equipo de desarrollo
- Revisar la documentación de la API

## 🔄 Versiones

- **v1.0.0**: Versión inicial con funcionalidades básicas de notificaciones
- **v1.1.0**: Agregado soporte para colas SQS y procesamiento asíncrono
- **v1.2.0**: Agregado soporte para plantillas de notificaciones

