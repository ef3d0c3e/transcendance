FROM debian:trixie

RUN apt-get update \
 && apt-get install -y --no-install-recommends \
    bash \
    tini \
    wget \
    curl \
    iproute2 \
    npm \
    make \
    libwebp-dev \
    gcc \
    libc6-dev \
 && rm -rf /var/lib/apt/lists/*

# Install Go
ARG GO_VERSION=1.27.1
RUN wget -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -O /tmp/go.tar.gz \
 && tar -xzf /tmp/go.tar.gz -C /opt \
 && rm /tmp/go.tar.gz

ENV PATH="/opt/go/bin:${PATH}"
ENV CGO_ENABLED=1

# Application directory
WORKDIR /opt/transcendance

# Install frontend dependencies first for Docker layer caching
COPY repo/frontend/package.json repo/frontend/package-lock.json ./frontend/
RUN cd frontend && npm ci

# Copy application source
COPY repo/ ./

# Build frontend during image construction
RUN make frontend

COPY service/setup.sh /opt/setup.sh
RUN chmod +x /opt/setup.sh

EXPOSE 8080

ENTRYPOINT ["/usr/bin/tini", "--", "/opt/setup.sh"]
