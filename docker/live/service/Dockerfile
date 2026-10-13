FROM debian:trixie

ARG USER_ID=1000
ARG GROUP_ID=1000

RUN apt-get update \
 && apt-get install -y --no-install-recommends \
    bash tini wget curl iproute2 npm make \
    libwebp-dev gcc libc6-dev \
 && rm -rf /var/lib/apt/lists/*

# Install Go
ARG GO_VERSION=1.27.1
RUN wget -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" \
      -O /tmp/go.tar.gz \
 && tar -xzf /tmp/go.tar.gz -C /opt \
 && rm /tmp/go.tar.gz

ENV PATH="/opt/go/bin:${PATH}"
ENV CGO_ENABLED=1

# Create a user matching the host UID/GID
RUN groupadd --gid "${GROUP_ID}" app \
 && useradd --uid "${USER_ID}" --gid "${GROUP_ID}" \
      --create-home --shell /bin/bash app \
 && mkdir -p /etc/nginx/ssl /opt/transcendance \
 && chown -R app:app /opt/transcendance \
 && chown app:app /etc/nginx/ssl

WORKDIR /opt/transcendance

# Copy the rest of the source
COPY --chown=app:app repo/ ./

# Copy the startup script
COPY service/setup.sh /opt/setup.sh
RUN chmod 755 /opt/setup.sh

USER app

EXPOSE 8080
ENTRYPOINT ["/usr/bin/tini", "--", "/opt/setup.sh"]

