# VERSION
VERSION ?= $(shell git describe --abbrev=0)+hash.$(shell git rev-parse --short HEAD)


DESTDIR    ?=
PREFIX     ?= /usr/local
EXECUTABLE := dyndns-client

# CONTAINER_RUNTIME
CONTAINER_RUNTIME ?= $(shell which docker)

# BUILD_IMAGE
BUILD_IMAGE_REGISTRY_HOST   := docker.io
BUILD_IMAGE_NAMESPACE       := volkerraschek
BUILD_IMAGE_REPOSITORY      := build-image
BUILD_IMAGE_VERSION         := latest
BUILD_IMAGE_FULLY_QUALIFIED := ${BUILD_IMAGE_REGISTRY_HOST}/${BUILD_IMAGE_NAMESPACE}/${BUILD_IMAGE_REPOSITORY}:${BUILD_IMAGE_VERSION:v%=%}

# BASE_IMAGE
BASE_IMAGE_REGISTRY_HOST    := docker.io
BASE_IMAGE_NAMESPACE        := library
BASE_IMAGE_REPOSITORY       := alpine
BASE_IMAGE_VERSION          := 3.12.0
BASE_IMAGE_FULLY_QUALIFIED  := ${BASE_IMAGE_REGISTRY_HOST}/${BASE_IMAGE_NAMESPACE}/${BASE_IMAGE_REPOSITORY}:${BASE_IMAGE_VERSION:v%=%}

# CONTAINER_IMAGE
CONTAINER_IMAGE_REGISTRY_HOST   := docker.io
CONTAINER_IMAGE_REGISTRY_USER   := volkerraschek
CONTAINER_IMAGE_NAMESPACE       := ${CONTAINER_IMAGE_REGISTRY_USER}
CONTAINER_IMAGE_REPOSITORY      := ${EXECUTABLE}
CONTAINER_IMAGE_VERSION         := latest
CONTAINER_IMAGE_FULLY_QUALIFIED := ${CONTAINER_IMAGE_REGISTRY_HOST}/${CONTAINER_IMAGE_NAMESPACE}/${CONTAINER_IMAGE_REPOSITORY}:${CONTAINER_IMAGE_VERSION:v%=%}
CONTAINER_IMAGE_UNQUALIFIED     := ${CONTAINER_IMAGE_NAMESPACE}/${CONTAINER_IMAGE_REPOSITORY}:${CONTAINER_IMAGE_VERSION:v%=%}

# BINARIES
# ==============================================================================
${EXECUTABLE}: clean bin/tmp/${EXECUTABLE}

bin/linux/amd64/$(EXECUTABLE):
	CGO_ENABLED=0 \
	GONOPROXY=$(shell go env GONOPROXY) \
	GONOSUMDB=$(shell go env GONOSUMDB) \
	GOPRIVATE=$(shell go env GOPRIVATE) \
	GOPROXY=$(shell go env GOPROXY) \
	GOSUMDB=$(shell go env GOSUMDB) \
	GOOS=linux \
	GOARCH=amd64 \
		go build -ldflags "-X main.version=${VERSION:v%=%}" -o ${@}

bin/tmp/${EXECUTABLE}:
	CGO_ENABLED=0 \
	GONOPROXY=$(shell go env GONOPROXY) \
	GONOSUMDB=$(shell go env GONOSUMDB) \
	GOPRIVATE=$(shell go env GOPRIVATE) \
	GOPROXY=$(shell go env GOPROXY) \
	GOSUMDB=$(shell go env GOSUMDB) \
		go build -ldflags "-X main.version=${VERSION:v%=%}" -o ${@}

# TEST
# ==============================================================================
PHONY+=test
test: clean bin/tmp/${EXECUTABLE}
	go test -v ./pkg/...

# CLEAN
# ==============================================================================
PHONY+=clean
clean:
	rm --force ${EXECUTABLE} || true
	rm --force --recursive bin || true

# CONTAINER IMAGE
# ==============================================================================
container-image/build:
	${CONTAINER_RUNTIME} build \
		--build-arg BASE_IMAGE=${BASE_IMAGE_FULLY_QUALIFIED} \
		--build-arg BUILD_IMAGE=${BUILD_IMAGE_FULLY_QUALIFIED} \
		--build-arg EXECUTABLE=${EXECUTABLE} \
		--build-arg GONOPROXY=$(shell go env GONOPROXY) \
		--build-arg GONOSUMDB=$(shell go env GONOSUMDB) \
		--build-arg GOPRIVATE=$(shell go env GOPRIVATE) \
		--build-arg GOPROXY=$(shell go env GOPROXY) \
		--build-arg GOSUMDB=$(shell go env GOSUMDB) \
		--build-arg VERSION=${VERSION:v%=%} \
		--no-cache \
		--tag ${CONTAINER_IMAGE_FULLY_QUALIFIED} \
		--tag ${CONTAINER_IMAGE_UNQUALIFIED} \
		.

container-image/push: container-image/build
	${CONTAINER_RUNTIME} login ${CONTAINER_IMAGE_REGISTRY_HOST} --username ${CONTAINER_IMAGE_REGISTRY_USER} --password ${CONTAINER_IMAGE_REGISTRY_PASSWORD}
	${CONTAINER_RUNTIME} push ${CONTAINER_IMAGE_FULLY_QUALIFIED}

# CONTAINER RUN - TEST
# ==============================================================================
PHONY+=container-run/test
container-run/test:
	$(MAKE) container-run COMMAND=${@:container-run/%=%}

# CONTAINER RUN - CLEAN
# ==============================================================================
PHONY+=container-run/clean
container-run/clean:
	$(MAKE) container-run COMMAND=${@:container-run/%=%}

# CONTAINER RUN - COMMAND
# ==============================================================================
PHONY+=container-run
container-run:
	${CONTAINER_RUNTIME} run \
		--env GONOPROXY=$(shell go env GONOPROXY) \
		--env GONOSUMDB=$(shell go env GONOSUMDB) \
		--env GOPRIVATE=$(shell go env GOPRIVATE) \
		--env GOPROXY=$(shell go env GOPROXY) \
		--env GOSUMDB=$(shell go env GOSUMDB) \
		--env EPOCH=${EPOCH} \
		--env VERSION=${VERSION:v%=%} \
		--env RELEASE=${RELEASE} \
		--rm \
		--volume $(shell pwd):/workspace \
			${BUILD_IMAGE_FULLY_QUALIFIED} \
				make ${COMMAND} \

# UN/INSTALL
# ==============================================================================
PHONY+=install
install: bin/tmp/${EXECUTABLE}
	install --directory ${DESTDIR}${PREFIX}/bin
	install --mode 755 bin/tmp/${EXECUTABLE} ${DESTDIR}${PREFIX}/bin/${EXECUTABLE}

	install --directory ${DESTDIR}/usr/lib/systemd/system
	install --mode 644 systemd/${EXECUTABLE}.service ${DESTDIR}/usr/lib/systemd/system
	install --mode 644 systemd/${EXECUTABLE}-docker.service ${DESTDIR}/usr/lib/systemd/system

	install --directory ${DESTDIR}/usr/share/licenses/${EXECUTABLE}
	install --mode 644 LICENSE ${DESTDIR}/usr/share/licenses/${EXECUTABLE}/LICENSE

PHONY+=uninstall
uninstall:
	-rm --recursive --force \
		${DESTDIR}${PREFIX}/bin/${EXECUTABLE} \
		${DESTDIR}/usr/lib/systemd/system/${EXECUTABLE}.service \
		${DESTDIR}/usr/lib/systemd/system/${EXECUTABLE}-docker.service \
		${DESTDIR}/usr/share/licenses/${EXECUTABLE}/LICENSE

# PHONY
# ==============================================================================
.PHONY: ${PHONY}
