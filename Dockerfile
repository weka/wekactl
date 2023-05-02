FROM golang:1.20.4-alpine3.17 as go-builder
# https://stackoverflow.com/questions/36279253/go-compiled-binary-wont-run-in-an-alpine-docker-container-on-ubuntu-host
RUN apk add --no-cache libc6-compat bash util-linux zip
COPY go.mod /src/go.mod
COPY go.sum /src/go.sum
WORKDIR /src
RUN go mod download
ADD . /src
ARG WEKACTL_AWS_LAMBDAS_BUCKETS
ARG LAMBDAS_ID
ARG AWS_DIST
ARG BUILD_VERSION
ARG COMMIT
RUN go run scripts/codegen/lambdas/gen_lambdas.go "$WEKACTL_AWS_LAMBDAS_BUCKETS" "$LAMBDAS_ID" "$AWS_DIST"
RUN chmod +x ./scripts/build_lambdas.sh
RUN ./scripts/build_lambdas.sh
RUN chmod +x ./scripts/build_wekactl.sh
RUN ./scripts/build_wekactl.sh "$BUILD_VERSION" "$COMMIT"
