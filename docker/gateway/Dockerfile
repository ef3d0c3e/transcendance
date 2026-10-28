FROM nginx:1.29-alpine

RUN apk add --no-cache openssl \
 && rm /etc/nginx/conf.d/default.conf

# Runs before the templates are rendered
COPY 05-certificate.sh /docker-entrypoint.d/
COPY templates/ /etc/nginx/templates/
COPY snippets/ /etc/nginx/snippets/

HEALTHCHECK --interval=15s --timeout=3s --retries=3 \
    CMD curl -fsS http://127.0.0.1:8081/healthz || exit 1
