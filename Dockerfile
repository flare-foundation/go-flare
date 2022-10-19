FROM golang:1.18 AS build

RUN apt-get update -y && \
    apt-get install -y rsync

WORKDIR /app/

COPY . ./
RUN ls -la && ls -la .git
RUN git init

WORKDIR /app/avalanchego/

# Can be one of following: leveldb, rocksdb, memdb. memdb
# docs: https://docs.avax.network/nodes/maintain/avalanchego-config-flags#database
ARG DB_TYPE=leveldb

# Build with RocksDB if enabled in build time
RUN if [ "$DB_TYPE" = "rocksdb" ]; \
        then export ROCKSDBALLOWED=1; \
        else unset ROCKSDBALLOWED; \
    fi; \
    /app/avalanchego/scripts/build.sh


FROM ubuntu:20.04

WORKDIR /app

# Can be one of following: leveldb, rocksdb, memdb. memdb
# docs: https://docs.avax.network/nodes/maintain/avalanchego-config-flags#database
ARG DB_TYPE=leveldb

ENV FBA_VALs=/app/conf/coston/fba_validators.json \
    HTTP_HOST=0.0.0.0 \
    HTTP_PORT=9650 \
    STAKING_PORT=9651 \
    PUBLIC_IP= \
    DB_DIR=/app/db \
    DB_TYPE=${DB_TYPE} \
    BOOTSTRAP_IPS= \
    BOOTSTRAP_IDS= \
    CHAIN_CONFIG_DIR=/app/conf \
    LOG_DIR=/app/logs \
    LOG_LEVEL=info \
    NETWORK_ID=costwo \
    AUTOCONFIGURE_PUBLIC_IP=1 \
    AUTOCONFIGURE_BOOTSTRAP=1 \
    AUTOCONFIGURE_BOOTSTRAP_ENDPOINT=https://coston2.flare.network/ext/info \
    EXTRA_ARGUMENTS=""

RUN apt-get update -y && \
    apt-get install -y curl jq

RUN mkdir -p /app/conf/coston /app/conf/C /app/logs /app/db

COPY --from=build /app/avalanchego/build /app/build
COPY entrypoint.sh /app/entrypoint.sh

EXPOSE ${STAKING_PORT}
EXPOSE ${HTTP_PORT}

VOLUME [ ${DB_DIR} ]
VOLUME [ ${LOG_DIR} ]
VOLUME [ ${CHAIN_CONFIG_DIR} ]

HEALTHCHECK CMD curl --fail http://localhost:${HTTP_PORT}/ext/health || exit 1

ENTRYPOINT [ "/usr/bin/bash" ]
CMD [ "/app/entrypoint.sh" ]
