ARG BASE_IMAGE
ARG BUILD_IMAGE

# BUILD
# ===========================================
FROM ${BUILD_IMAGE} AS build
ADD . /workspace

ARG EXECUTABLE
ARG GONOPROXY
ARG GONOSUMDB
ARG GOPRIVATE
ARG GOPROXY
ARG GOSUMDB
ARG VERSION

RUN make bin/linux/amd64/${EXECUTABLE}

# TARGET CONTAINER
# ===========================================
FROM ${BASE_IMAGE}

ARG EXECUTABLE

RUN apk add --update bind-tools
COPY --from=build /workspace/bin/linux/amd64/${EXECUTABLE} /usr/bin/${EXECUTABLE}
ENTRYPOINT [ "/usr/bin/dyndns-client" ]