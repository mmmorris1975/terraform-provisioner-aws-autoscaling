EXE  := terraform-provisioner-aws-autoscaling
VER  := $(shell git describe --tags)
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

.PHONY: release darwin linux windows clean

$(EXE): Gopkg.lock *.go
	go build -v -o $@

Gopkg.lock: Gopkg.toml
	dep ensure

release: $(EXE) darwin windows linux

darwin linux:
	GOOS=$@ go build -o $(EXE)_v$(VER)_$@-$(GOARCH)

windows:
	GOOS=$@ go build -o $(EXE)_v$(VER)_$@-$(GOARCH).exe

clean:
	rm -f $(EXE) $(EXE)-v*-*-*
