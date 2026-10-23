#checkov:skip=CKV_DOCKER_3: "Ensure that a user for the container has been created"
FROM golang:1.26.10-trixie@sha256:909771fef53e3dcb0950255f0c9a8b4728c3b841e61509c7b29b098de7ee02d4 AS build

WORKDIR /app/

COPY ./.git /app/.git
COPY ./avalanchego /app/avalanchego
COPY ./config /app/config

WORKDIR /app/avalanchego/

RUN /app/avalanchego/scripts/build.sh

FROM ubuntu:24.04@sha256:68ee22004a9109b44973313b02ff7b1de250f1de0f5ecb628d73f579aaff33ac

WORKDIR /app

ENV HTTP_HOST=0.0.0.0 \
    HTTP_PORT=9650 \
    STAKING_PORT=9651 \
    PUBLIC_IP= \
    DB_DIR=/app/db \
    DB_TYPE=leveldb \
    CHAIN_DATA_DIR=/app/db/chainData \
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

# hadolint ignore=DL3008
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates curl jq && \
    rm -rf /var/lib/apt/lists/* && \
    mkdir -p /app/conf/coston /app/conf/C /app/logs /app/db /app/db/chainData

COPY --from=build /app/avalanchego/build /app/build
COPY entrypoint.sh /app/entrypoint.sh

EXPOSE ${STAKING_PORT}
EXPOSE ${HTTP_PORT}

VOLUME [ "${DB_DIR}" ]
VOLUME [ "${LOG_DIR}" ]
VOLUME [ "${CHAIN_CONFIG_DIR}" ]

HEALTHCHECK CMD ["/bin/sh", "-c", "curl --fail http://localhost:${HTTP_PORT}/ext/health || exit 1"]

ENTRYPOINT [ "/usr/bin/bash" ]
CMD [ "/app/entrypoint.sh" ]