# Build stage
FROM golang:1.26-alpine AS builder

# Set Go proxy for China
ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /app

# Copy go mod files
COPY go.mod ./

# Copy source code
COPY . .

# Download dependencies and build
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o main .

# Run stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

# Set timezone to Asia/Shanghai
ENV TZ=Asia/Shanghai

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/main .

# Copy default skills
COPY --from=builder /app/skills ./skills

# Create skills directory if not exists
RUN mkdir -p /app/skills

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]
