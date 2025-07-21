# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
RUN adduser -D -u 10001 scratchuser
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o printloop .
RUN mkdir -p /app/files/uploads /app/files/results /app/tmp && \
    chown -R scratchuser:scratchuser /app/files && \
    chown -R scratchuser:scratchuser /app/tmp

FROM scratch
COPY --from=builder /app/printloop /printloop
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder --chown=10001:10001 /app/files /files
COPY --from=builder --chown=10001:10001 /app/tmp /tmp
USER scratchuser
EXPOSE 8080
ENTRYPOINT ["/printloop"]