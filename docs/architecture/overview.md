# Architecture Overview

## System Architecture

The YouTube Webhook Service is a serverless application built on Google Cloud Functions that processes YouTube PubSubHubbub notifications and triggers GitHub Actions workflows.

```
YouTube → PubSubHubbub → Cloud Function → GitHub API → Actions Workflow → Website Update
```

## Core Components

### 1. Webhook Handler (`webhook.go`)
- Entry point for all HTTP requests
- Routes requests to appropriate handlers
- Manages PubSubHubbub verification challenges

### 2. Subscription Management
- Manages YouTube channel subscriptions via PubSubHubbub
- Persists subscription state in Cloud Storage
- Handles auto-renewal via Cloud Scheduler

### 3. Notification Processing
- Processes incoming YouTube feed notifications
- Filters for new video publications
- Triggers GitHub workflow dispatches

### 4. Storage Layer
- Cloud Storage for persistent state
- Optimized with caching and connection pooling
- Thread-safe concurrent access

## Architectural Patterns

### Clean Architecture
The codebase follows Clean Architecture principles with clear separation of concerns:

```
├── Domain Layer (Models & Business Logic)
│   ├── Subscription
│   ├── SubscriptionState
│   └── Notification
├── Application Layer (Services)
│   ├── NotificationService
│   ├── StorageService
│   └── GitHubClient
└── Infrastructure Layer
    ├── Cloud Storage
    ├── PubSubHubbub
    └── GitHub API
```

### Service-Oriented Architecture (SOA)
Each service has a single responsibility:
- `NotificationService`: Process YouTube notifications
- `StorageService`: Manage persistent state
- `GitHubClient`: Interact with GitHub API
- `VideoProcessor`: Parse and validate video data

### Event-Driven Architecture
The system is event-driven with multiple trigger points:
- YouTube publishes video → PubSubHubbub notification
- Notification received → GitHub workflow dispatch
- Subscription expiring → Auto-renewal trigger

### Domain-Driven Design (DDD)
Rich domain models encapsulate business logic:
- `Subscription`: Manages subscription lifecycle
- `SubscriptionState`: Aggregates all subscriptions
- `Feed`: Represents YouTube Atom feeds

## Design Principles

### SOLID Principles
- **Single Responsibility**: Each service has one reason to change
- **Open/Closed**: Services are extensible without modification
- **Liskov Substitution**: Interfaces enable substitutable implementations
- **Interface Segregation**: Small, focused interfaces
- **Dependency Inversion**: Depend on abstractions, not concretions

### Infrastructure as Code
All infrastructure is defined in Terraform:
- Reproducible deployments
- Version controlled infrastructure
- Environment-specific configurations

### Security by Design
- Service account authentication
- Least privilege access
- No hardcoded secrets
- Input validation at all boundaries

## Scalability Considerations

### Serverless Architecture
- Auto-scaling with Cloud Functions Gen 2
- Pay-per-use pricing model
- No server management overhead

### Connection Pooling
- Reuses Cloud Storage connections
- Reduces latency and resource usage
- Singleton pattern for client instances

### Caching Strategy
- 5-minute TTL for subscription state
- Reduces Cloud Storage API calls
- Thread-safe cache implementation

### Concurrent Processing
- Mutex-protected shared state
- Deep copying for data isolation
- Race-condition-free operations

## Monitoring & Observability

### Structured Logging
- Consistent log format
- Contextual information
- Error tracking with stack traces

### Health Checks
- Subscription status monitoring
- Renewal success tracking
- API availability checks

### Metrics
- Active subscription count
- Renewal success rate
- Notification processing time
- API call latency

## Future Enhancements

### Phase 4: Monitoring & Alerting
- Cloud Monitoring integration
- Custom metrics and dashboards
- Alert policies for failures

### Potential Improvements
- Multi-region deployment
- Webhook signature verification
- Admin dashboard for management
- Batch subscription operations