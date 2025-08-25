FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Install iocgo and build for the target architecture
RUN make install

# Build for the target architecture using TARGETARCH
ARG TARGETARCH
RUN if [ "$TARGETARCH" = "arm64" ]; then \
        make build-linux-arm64; \
    else \
        make build-linux; \
    fi

FROM alpine:3.19.0

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy the binary based on architecture
ARG TARGETARCH
COPY --from=builder /app/bin/linux/server* ./server

# Create config file
RUN echo -e 'server:\n  port: 8080' > ./config.yaml && \
    chmod +x server

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

USER appuser

EXPOSE 8080

ENV ARG "--port 8080"
ENTRYPOINT ["sh", "-c", "./server ${ARG}"]