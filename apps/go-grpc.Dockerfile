FROM golang AS dev
ARG APP
ARG GIT_COMMIT
ARG GIT_REPO
ARG VERSION
ARG GRPC_HEALTH_PROBE_VERSION=v0.4.37
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOCACHE=/go/.cache
ENV GRPC_HEALTH_PROBE_VERSION="${GRPC_HEALTH_PROBE_VERSION}"
USER 1000
WORKDIR /go/root
COPY . .
WORKDIR /go/root/apps/$APP
RUN go install ./... \
  && wget -qO /go/bin/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64 \
  && chmod +x /go/bin/grpc_health_probe
COPY --from=cosmtrek/air /go/bin/air /go/bin/air
CMD [ "air", "-build.stop_on_error", "true", "-build.send_interrupt", "true", "-build.rerun", "true" ]

FROM golang AS builder
ARG APP
ARG GIT_COMMIT
ARG GIT_REPO
ARG VERSION
WORKDIR /go/root
COPY . .
WORKDIR /go/root/apps/$APP
ENV CGO_ENABLED=0
ENV GOOS=linux
RUN go build \
  -ldflags \
  "-w -s -X $GIT_REPO/cmd.Version=$VERSION -X $GIT_REPO/cmd.GitCommit=$GIT_COMMIT" \
  -o /go/app
COPY --from=dev /go/bin/grpc_health_probe /bin/grpc_health_probe
ENTRYPOINT [ "/go/app" ]

FROM scratch
ARG GIT_COMMIT
ARG VERSION
ENV GIT_COMMIT="${GIT_COMMIT}"
ENV VERSION="${VERSION}"
WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /go/app /app
COPY --from=dev /go/bin/grpc_health_probe /bin/grpc_health_probe
ENTRYPOINT [ "/app/app" ]
