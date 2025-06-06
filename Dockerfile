FROM golang:tip-bullseye

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build and run the application
RUN go build -o app .

# Expose the port from .env
EXPOSE ${PORT:-8080}

# Run the application
CMD ["./app"]