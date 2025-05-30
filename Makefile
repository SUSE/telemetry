.DEFAULT_GOAL := build
LOG_LEVEL = info
CNTR_MGR = docker

-include Makefile.docker
include Makefile.golang
