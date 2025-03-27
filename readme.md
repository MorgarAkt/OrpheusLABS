# OrpheusLABS - AI-Driven Music Generation Suite

An intelligent music composition platform combining neural networks with modern web technologies, featuring a Tailwind-powered responsive interface.

## Features

- 🧠 Neural Network music generation
- 🔐 JWT Authentication with cookies
- 🎹 Interactive MIDI player 
- 📦 Modular architecture with clean separation
- 🚀 RabbitMQ-powered async task queue
- 📊 Admin dashboard with analytics
<!-- - 📱 Mobile-responsive design -->

## Tech Stack

**Core**
```go
Go 1.21.4 | PostgreSQL | RabbitMQ | Gin
```

**Frontend**
```js
Tailwind CSS 3.3 | HTMX | HTML5 Canvas
```

**AI Engine**
```python
Python 3.11 | PyTorch 2.0  | LSTM Networks | TRANSFORMER Networks
```

## Getting Started

### Prerequisites
- Go 1.21.4
- PostgreSQL 15+
- RabbitMQ 3.12+
- Node.js 18+ (for Tailwind processing)

### Installation
```bash
git clone https://github.com/morgarakt/OrpheusLABS.git
cd OrpheusLABS
go mod download

cd web
npm install
npx tailwindcss -i ./static/css/input.css -o ./static/css/style.css --watch
```

### Configuration (`.env`)
```ini
DB_HOST=localhost
DB_NAME=orpheuslabs
JWT_SECRET=your_secure_secret
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
```

## Development Workflow
```bash
# Go backend
go run cmd/main.go 

# AI worker
cd services/python_service
python midi_consumer.py

# Tailwind CSS watcher
cd web
npx tailwindcss -i ./static/css/input.css -o ./static/css/style.css --watch
```

## Key Components

| Directory          | Purpose                          |
|--------------------|----------------------------------|
| `internal/handlers`| HTTP request controllers         |
| `web/static/css`   | Tailwind-generated styles       |
| `services/python`  | AI model training/inference     |
| `internal/utils`   | JWT, DB, middleware utilities   |

## Tailwind Architecture

```bash
web/
├── tailwind.config.js
├── static/css/
│   ├── input.css    # Tailwind directives
│   └── style.css    # Generated output
└── templates/       # HTML components
```
<!-- 
## API Documentation

[![Run in Postman](https://run.pstmn.io/button.svg)](https://god.gw.postman.com/run-collection/your-collection-id)

**Sample Endpoint**:
```http
POST /api/v1/generate
Content-Type: application/json

{
  "genre": "classical",
  "duration": 120,
  "complexity": 0.7
}
```

## License

Apache 2.0 License - See [LICENSE](LICENSE) for details.

--- -->

**Screenshot Preview**  
![OrheusLabs Interface](./.github/screencast.GIF)
```

