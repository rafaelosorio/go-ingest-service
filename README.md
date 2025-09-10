
# go-ingest-service

Minimal **Go microservice** for event ingestion with health checks, structured logging, and Prometheus metrics.  
Built to demonstrate **system design, observability, and scalability** best practices.


## ✨ Features
- REST API using [chi](https://github.com/go-chi/chi)
- In-memory event storage (easy to extend to PostgreSQL/Redis)
- Health check endpoint (`/healthz`)
- Structured logging with [zerolog](https://github.com/rs/zerolog)
- Prometheus metrics (`/metrics`)
- Graceful shutdown with `context.Context`
- Ready for Docker and CI/CD

## 📦 Installation

Clone the repo:

```bash
git clone https://github.com/rafaelosorio/go-ingest-service.git
cd go-ingest-service
```

Install dependencies:

```bash
go mod tidy
```

Run the service:

```bash
go run ./cmd/api
```

## 🚀 Usage

### Create an event
```bash
curl -XPOST localhost:8080/events   -H "Content-Type: application/json"   -d '{"type":"signup","payload":"{\"user_id\":123}"}'
```

### List events
```bash
curl localhost:8080/events
```

### Health check
```bash
curl localhost:8080/healthz
```

### Metrics
```bash
curl localhost:8080/metrics | head
```

---

## 🛠 Project Structure

```
go-ingest-service/
 └── cmd/api/         # main entrypoint
     └── main.go
```

- `Store` is an in-memory implementation (can be replaced with Postgres).
- Handlers include basic CRUD-like endpoints.
- Middleware handles logging, request IDs, recoveries, and timeouts.

---

## 📊 Observability

The service exposes **Prometheus metrics**:
- `http_requests_total` (by route/method/code)
- `http_request_duration_seconds` (latency histogram)

Logs are structured with zerolog:
```
{"level":"info","method":"POST","path":"/events","duration":0.001,"time":"2025-03-01T12:00:00Z","message":"request"}
```

---

## 🧪 Next Steps

- [ ] Add PostgreSQL persistence layer  
- [ ] Add Dockerfile & docker-compose  
- [ ] Add OpenTelemetry tracing  
- [ ] Add k6/vegeta load testing scripts  
- [ ] Deploy example (Kubernetes)  

---

## 📜 License
MIT
