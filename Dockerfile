FROM golang:1.23-alpine AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN GOOS=linux go build -o pr-reviewer-assignment-service

FROM alpine:latest
WORKDIR /app

COPY --from=build /app/pr-reviewer-assignment-service .

EXPOSE 8080
CMD ["./pr-reviewer-assignment-service"]