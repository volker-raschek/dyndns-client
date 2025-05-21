# VERSION
VERSION ?= $(shell git describe --abbrev=0)+hash.$(shell git rev-parse --short HEAD)

DESTDIR ?=
PREFIX ?= /usr/local
EXECUTABLE := dyndns-client

# BINARIES
# ==============================================================================
all: ${EXECUTABLE}

${EXECUTABLE}:
	CGO_ENABLED=0 \
	GOPRIVATE=$(shell go env GOPRIVATE) \
	GOPROXY=$(shell go env GOPROXY) \
		go build -ldflags "-X main.version=${VERSION:v%=%}" -o ${@}

# TEST
# ==============================================================================
PHONY+=test/unit
test/unit: clean ${EXECUTABLE}
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
install: ${EXECUTABLE}
	install --directory ${DESTDIR}${PREFIX}/bin
	install --mode 755 ${EXECUTABLE} ${DESTDIR}${PREFIX}/bin/${EXECUTABLE}

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
