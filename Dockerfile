# --------- frontend -----------
FROM node:20.10-alpine as frontendBuilder

WORKDIR /app

ARG proxy

RUN npm install -g pnpm@8.14.0

COPY ./web/pnpm-lock.yaml /app/web/pnpm-lock.yaml
COPY ./web/package.json /app/web/package.json
RUN cd /app/web/ && pnpm i
COPY ./web /app/web
RUN cd /app/web/ && pnpm build

COPY ./pal-conf/pnpm-lock.yaml /app/pal-conf/pnpm-lock.yaml
COPY ./pal-conf/package.json /app/pal-conf/package.json
RUN cd /app/pal-conf/ && pnpm i
COPY ./pal-conf /app/pal-conf
RUN cd /app/pal-conf/ && pnpm build

RUN mv /app/pal-conf/dist/assets/* /app/assets
RUN mv /app/pal-conf/dist/index.html /app/pal-conf.html

# --------- sav_cli -----------
FROM python:3.11-alpine as savBuilder

WORKDIR /app

ARG proxy
ARG TARGETARCH
ARG version
ARG assets_version

RUN apk update && apk add curl unzip
COPY ./script/download-release-asset.sh /app/script/download-release-asset.sh
RUN chmod +x /app/script/download-release-asset.sh
RUN mkdir -p /app/dist && \
    ASSET_VERSION="${assets_version:-${version:-v0.9.9}}" && \
    export PST_RELEASE_VERSION="$ASSET_VERSION" && \
    if [ "$TARGETARCH" = "amd64" ]; then \
        asset_name="sav_cli_linux_x86_64"; \
    elif [ "$TARGETARCH" = "arm64" ]; then \
        asset_name="sav_cli_linux_aarch64"; \
    else \
        echo "Unsupported architecture: $TARGETARCH" && exit 1; \
    fi && \
    /app/script/download-release-asset.sh "$asset_name" /app/dist/sav_cli
RUN chmod +x /app/dist/sav_cli

# --------- map tiles -----------
FROM python:3.11-alpine as mapDownloader

WORKDIR /app

ARG version
ARG assets_version

RUN apk update && apk add curl unzip
COPY ./script/download-release-asset.sh /app/script/download-release-asset.sh
RUN chmod +x /app/script/download-release-asset.sh
RUN ASSET_VERSION="${assets_version:-${version:-v0.9.9}}" && \
    export PST_RELEASE_VERSION="$ASSET_VERSION" && \
    /app/script/download-release-asset.sh map.zip /app/map.zip && \
    unzip /app/map.zip -d /app

# --------- backend -----------
FROM golang:1.21-alpine as backendBuilder

ARG proxy
ARG version

WORKDIR /app
ADD . .

COPY --from=frontendBuilder /app/assets /app/assets
COPY --from=frontendBuilder /app/index.html /app/index.html
COPY --from=frontendBuilder /app/pal-conf.html /app/pal-conf.html
COPY --from=mapDownloader /app/map /app/map

RUN if [ ! -z "$proxy" ]; then \
    export GOPROXY=https://goproxy.io,direct && \
    go build -ldflags="-s -w -X 'main.version=${version}'" -o /app/dist/pst main.go; \
    else \
    go build -ldflags="-s -w -X 'main.version=${version}'" -o /app/dist/pst main.go; \
    fi

# --------- runtime -----------
FROM frolvlad/alpine-glibc as runtime

WORKDIR /app

ENV SAVE__DECODE_PATH /app/sav_cli

COPY --from=savBuilder /app/dist/sav_cli /app/sav_cli
COPY --from=backendBuilder /app/dist/pst /app/pst

EXPOSE 8080

CMD ["/app/pst"]
