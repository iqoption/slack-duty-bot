all: make

BUILD_OS:=linux
BUILD_ARCH:=amd64

dep-install:
	go get -v -u github.com/golang/dep/cmd/dep
dep-ensure:
	dep ensure
build:
	GOOS=${BUILD_OS} GOARCH=${BUILD_ARCH} go build -v
make: dep-install dep-ensure build
