# Seta Training: Microservices & Domain Concurrency

This repository contains a robust backend system built for managing Users, Teams, and Digital Assets, implementing Clean Architecture and modern observability practices.

---

## Tech Stack & Rationale

| Layer | Technology | Rationale | Trade-offs |
| :--- | :--- | :--- | :--- |
| **Core** | **Go (Golang)** | Designed for high concurrency (goroutines) and cloud-native performance. | More boilerplate (error handling) than high-level languages. |
| **Framework**| **Gin** | Lightweight, high-performance, and provides a great developer experience. | Not part of the standard library. |
| **Database** | **PostgreSQL** | Reliable relational storage for users, teams, and asset hierarchies. | Complex schema migrations compared to NoSQL. |
| **Auth** | **JWT + Bcrypt** | Stateless authentication and secure password hashing. | Revoking individual tokens requires a blacklist mechanism (Redis). |
| **Docs** | **Swagger (Swaggo)** | Automated API documentation generation from code comments. | Maintenance overhead for keeping comments in sync. |
| **Monitoring** | **Prometheus** | Real-time metrics collection via the `/metrics` endpoint. | Time-series data can grow quickly in storage. |
| **Logging** | **Loki + Promtail** | Centralized, cost-effective log aggregation with structured JSON. | Requires running additional background containers. |
| **Visualization**| **Grafana** | Unified dashboards for metrics and log exploration. | Initial dashboard setup overhead. |

---

## System Architecture (Clean Architecture)

The project follows a modular structure based on **Clean Architecture**:
- `domain/`: Core business entities and interfaces.
- `usecase/`: Pure business logic (orchestrating repositories and domain rules).
- `repository/`: Data persistence implementations (GORM/PostgreSQL).
- `delivery/`: External entry points (Gin HTTP handlers & Middleware).
- `infrastructure/`: External tools (Database connection, Logger setup).

---

## Getting Started

### Prerequisites
- [Go 1.24+](https://golang.org/dl/)
- [Docker & Docker Compose](https://www.docker.com/products/docker-desktop)

### 1. Setup Infrastructure
Start the database and observability stack (PostgreSQL, Prometheus, Loki, Grafana):
```bash
docker compose up -d
```

### 2. Run the Application
```bash
go run cmd/api/main.go
```
The server will start on `http://localhost:3000`.

---

## Observability & API Documentation

- **Swagger UI:** [http://localhost:3000/swagger/index.html](http://localhost:3000/swagger/index.html)
- **Grafana (Dashboards):** [http://localhost:3001](http://localhost:3001) (User: `admin` / Pass: `admin`)
- **Prometheus:** [http://localhost:9090](http://localhost:9090)
- **Loki (Logs):** Queryable via the **Explore** tab in Grafana using datasource `Loki`.

---

## Testing the API

### Automated Test Script
We've provided a script to test the core flows (Register -> Login -> Team Mgmt -> Assets):
```bash
./test_api.sh
```

### Concurrency Challenge: Bulk User Import
To test high-speed concurrent user creation, upload a CSV file to the import endpoint:
```bash
curl -X POST http://localhost:3000/api/v1/users/import \
  -H "Authorization: Bearer <TOKEN>" \
  -F "file=@users.csv"
```
*Note: The system uses a **Worker Pool** of goroutines to process rows in parallel.*

---

## Roadmap & Stages

- `[x]` **Stage 1:** Identity & Teams (Auth, RBAC, Team Management).
- `[x]` **Stage 2:** Domain & Concurrency (Folders, Notes, Sharing, Bulk Import).
- `[x]` **Stage 3:** Observability (Metrics, Logging, Visualization).
- `[ ]` **Stage 4 (Next):** Scaling (Message Queues, Redis Caching).
