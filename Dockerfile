# Stage 1: Build the Go binary
FROM golang:1.25-bookworm AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /chromerpc ./cmd/chromerpc

# Stage 2: Runtime with Chrome
FROM debian:bookworm-slim

# Install Chrome and dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    wget \
    gnupg \
    ca-certificates \
    fonts-liberation \
    libasound2 \
    libatk-bridge2.0-0 \
    libatk1.0-0 \
    libcups2 \
    libdbus-1-3 \
    libdrm2 \
    libgbm1 \
    libgtk-3-0 \
    libnspr4 \
    libnss3 \
    libx11-xcb1 \
    libxcomposite1 \
    libxdamage1 \
    libxrandr2 \
    xdg-utils \
    && wget -q -O - https://dl.google.com/linux/linux_signing_key.pub | gpg --dearmor -o /usr/share/keyrings/google-chrome.gpg \
    && echo "deb [arch=amd64 signed-by=/usr/share/keyrings/google-chrome.gpg] http://dl.google.com/linux/chrome/deb/ stable main" > /etc/apt/sources.list.d/google-chrome.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends google-chrome-stable \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /chromerpc /usr/local/bin/chromerpc

# Chrome needs a writable home directory
RUN useradd -m -s /bin/bash chromerpc
USER chromerpc
WORKDIR /home/chromerpc

# gRPC port
EXPOSE 50051

ENTRYPOINT ["chromerpc"]
CMD ["--headless", "--addr", ":50051"]
