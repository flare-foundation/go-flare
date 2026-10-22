FROM golang:1.26.8-trixie@sha256:aae9c439b447d24c93ee483408992857cf39a3fb6ab7fe53affc65d834891fa8 AS build

WORKDIR /app/

COPY ./.git /app/.git
COPY ./avalanchego /app/avalanchego
COPY ./config /app/config

WORKDIR /app/avalanchego/

RUN /app/avalanchego/scripts/build.sh

FROM ubuntu:24.04

WORKDIR /app

ENV HTTP_HOST=0.0.0.0 \
    HTTP_PORT=9650 \
    STAKING_PORT=9651 \
    PUBLIC_IP= \
    DB_DIR=/app/db \
    DB_TYPE=leveldb \
    CHAIN_DATA_DIR=/app/chainData \
    BOOTSTRAP_IPS= \
    BOOTSTRAP_IDS= \
    CHAIN_CONFIG_DIR=/app/conf \
    LOG_DIR=/app/logs \
    LOG_LEVEL=info \
    NETWORK_ID=costwo \
    AUTOCONFIGURE_PUBLIC_IP=1 \
    AUTOCONFIGURE_BOOTSTRAP=1 \
    AUTOCONFIGURE_BOOTSTRAP_ENDPOINT=https://coston2-bootstrap.flare.network/ext/info \
    EXTRA_ARGUMENTS="" \
    BOOTSTRAP_BEACON_CONNECTION_TIMEOUT="1m" \
    HTTP_ALLOWED_HOSTS="*"

RUN apt-get update -y && \
    apt-get install -y curl jq

RUN mkdir -p /app/conf/coston /app/conf/C /app/logs /app/db /app/chainData

COPY --from=build /app/avalanchego/build /app/build
COPY entrypoint.sh /app/entrypoint.sh

EXPOSE ${STAKING_PORT}
EXPOSE ${HTTP_PORT}

VOLUME [ "${DB_DIR}" ]
VOLUME [ "${CHAIN_DATA_DIR}" ]
VOLUME [ "${LOG_DIR}" ]
VOLUME [ "${CHAIN_CONFIG_DIR}" ]

HEALTHCHECK CMD curl --fail http://localhost:${HTTP_PORT}/ext/health || exit 1

ENTRYPOINT [ "/usr/bin/bash" ]
CMD [ "/app/entrypoint.sh" ]