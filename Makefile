BINS:=godsect

all: $(BINS)

ME:=$(firstword $(MAKEFILE_LIST))

VERSION ?= $(GODSECT_VERSION)
ifeq ($(strip $(VERSION)),)
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev-$(shell date +%Y%m%d_%H%M%S))
endif
LDFLAGS := -X main.version=$(VERSION)

godsect: godsect.go $(ME)
	go build -ldflags "$(LDFLAGS)" -o godsect

clean:
	-@ [ -x godsect ] && rm godsect

check:
	@echo no checks yet

install:
	mkdir -p $(PREFIX)/bin
	install $(BINS) $(PREFIX)/bin
