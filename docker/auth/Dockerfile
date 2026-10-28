FROM golang:1.27.1-trixie AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY docker/auth ./docker/auth
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/auth ./docker/auth \
 && mkdir /out/data

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/auth /auth
# Host key, session secret and authorized_keys, owned by the nonroot user
COPY --from=build --chown=65532:65532 /out/data /data
ENV PATH=/
EXPOSE 2233 9000
HEALTHCHECK --interval=15s --timeout=3s --retries=3 CMD ["/auth", "health"]
ENTRYPOINT ["/auth"]
