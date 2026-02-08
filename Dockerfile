FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/collab-editor . && chmod 755 /app/collab-editor

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/collab-editor /collab-editor
COPY --from=builder /app/static /static
EXPOSE 8080
ENTRYPOINT ["/collab-editor"]
