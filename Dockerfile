FROM golang:1.26-alpine AS build
WORKDIR /workspace
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN go build -o /out/worker ./cmd/worker

FROM alpine:3.21
RUN adduser -D -g '' appuser
USER appuser
WORKDIR /app
COPY --from=build /out/worker /app/worker
EXPOSE 8084
ENTRYPOINT ["/app/worker"]
