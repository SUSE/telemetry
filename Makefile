export GO_COVERAGE_PROFILE = /tmp/.coverage.telemetry.out

.DEFAULT_GOAL := build

ifeq ($(MAKELEVEL),0)

LOG_LEVEL = info
CNTR_MGR = docker

-include Makefile.docker

SUBDIRS = \
  cmd/authenticator \
  cmd/clientds \
  cmd/generator \
  examples/app \
  .

TARGETS = fmt vet build build-only clean test test-clean test-verbose mod-tidy mod-download mod-update test-mod-update

.PHONY: $(TARGETS)

$(TARGETS)::
	$(foreach subdir, $(SUBDIRS), $(MAKE) -C $(subdir) $@ || exit 1;)

test-coverage: test
	go tool cover --func=$(GO_COVERAGE_PROFILE)

else
include Makefile.golang
endif
