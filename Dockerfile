# --- Stage 1: Build ---
FROM golang:1.23-bookworm AS builder

WORKDIR /app

# Install compilation dependencies and wget
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    wget \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Download and extract ONNX Runtime C/C++ libraries (Linux x64)
WORKDIR /onnxruntime
RUN wget https://github.com/microsoft/onnxruntime/releases/download/v1.18.1/onnxruntime-linux-x64-1.18.1.tgz \
    && tar -xzf onnxruntime-linux-x64-1.18.1.tgz --strip-components=1

# Copy dependencies list
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

# Copy source code files
COPY . .

# Build Go application linking ONNX C library
ENV CGO_ENABLED=1
ENV CGO_CFLAGS="-I/onnxruntime/include"
ENV CGO_LDFLAGS="-L/onnxruntime/lib -lonnxruntime"
RUN go build -ldflags="-s -w" -o car-price-api ./cmd/server/main.go

# --- Stage 2: Runtime ---
FROM debian:bookworm-slim AS runner

WORKDIR /app

# Install runtime dependencies (like OpenMP)
RUN apt-get update && apt-get install -y --no-install-recommends \
    libgomp1 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Copy the compiled executable
COPY --from=builder /app/car-price-api /app/car-price-api

# Copy the ONNX shared library from builder
COPY --from=builder /onnxruntime/lib/libonnxruntime.so.1.18.1 /usr/local/lib/libonnxruntime.so.1.18.1
RUN ln -s /usr/local/lib/libonnxruntime.so.1.18.1 /usr/local/lib/libonnxruntime.so \
    && ldconfig

# Set runtime environmental settings
ENV PORT=8000
ENV ONNXRUNTIME_SHARED_LIB_PATH=/usr/local/lib/libonnxruntime.so

EXPOSE 8000

CMD ["/app/car-price-api"]
