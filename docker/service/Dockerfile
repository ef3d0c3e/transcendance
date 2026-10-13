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
	libc6-dev \
	gcc \
 && rm -rf /var/lib/apt/lists/*

COPY setup.sh /opt/setup.sh

RUN chmod +x /opt/setup.sh

EXPOSE 8080

ENTRYPOINT ["/usr/bin/tini", "--", "/opt/setup.sh"]
