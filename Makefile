# VERSION
VERSION ?= $(shell git describe --abbrev=0)+hash.$(shell git rev-parse --short HEAD)

DESTDIR    ?=
PREFIX     ?= /usr/local
EXECUTABLE := dyndns-client

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

# UN/INSTALL
# ==============================================================================
PHONY+=install
install: bin/tmp/${EXECUTABLE}
	install --directory ${DESTDIR}${PREFIX}/bin
	install --mode 755 bin/tmp/${EXECUTABLE} ${DESTDIR}${PREFIX}/bin/${EXECUTABLE}

	install --directory ${DESTDIR}/usr/lib/systemd/system
	install --mode 644 systemd/${EXECUTABLE}.service ${DESTDIR}/usr/lib/systemd/system

	install --directory ${DESTDIR}/usr/share/licenses/${EXECUTABLE}
	install --mode 644 LICENSE ${DESTDIR}/usr/share/licenses/${EXECUTABLE}/LICENSE

PHONY+=uninstall
uninstall:
	-rm --recursive --force \
		${DESTDIR}${PREFIX}/bin/${EXECUTABLE} \
		${DESTDIR}/usr/lib/systemd/system/${EXECUTABLE}.service \
		${DESTDIR}/usr/share/licenses/${EXECUTABLE}/LICENSE

# PHONY
# ==============================================================================
.PHONY: ${PHONY}
