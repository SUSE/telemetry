ifeq ($(MAKELEVEL),0)

.DEFAULT_GOAL := build

SUBDIRS = \
  . \
  cmd/authenticator \
  cmd/clientds \
  cmd/generator \
  examples/app

TARGETS = fmt vet build build-only clean test test-clean test-verbose tidy

.PHONY: $(TARGETS)

$(TARGETS):
	$(foreach subdir, $(SUBDIRS), $(MAKE) -C $(subdir) $@;)
else
include Makefile.golang
endif
