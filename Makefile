all: build

.PHONY : all package

# prevent run if docker not found
ifeq (, $(shell which docker))
	$(error "Binary docker not found in $(PATH)")
endif

APP_NAME?=slack-duty-bot

# strict variables
override ROOT_DIR=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
ifneq (, $(shell which git))
override MOD_NAME=$(shell git config --get remote.origin.url | cut -c 5- | rev | cut -c 5- | rev | tr : / || consul-initializer)
endif
ifeq ($(MOD_NAME),)
override MOD_NAME=consul-initializer
endif

# vendor variables
FORCE_INIT=0
ifeq (,$(wildcard ./go.mod))
	FORCE_INIT=1
endif

FORCE_VENDOR=0
ifeq (,$(wildcard ./vendor/.*))
	FORCE_VENDOR=1
endif

# build go binary variables
GO_VERSION=1.11
GOOS?=$(shell go env GOOS || echo linux)
GOARCH?=$(shell go env GOARCH || echo amd64)
CGO_ENABLED?=0

# docker build variables
DOCKER_NAMESPACE?=
DOCKER_IMAGE=${DOCKER_NAMESPACE}/${APP_NAME}
DOCKER_TAG?=1.0.0
DOCKER_USER?=
DOCKER_PASSWORD?=

init:
ifeq ($(FORCE_INIT), 1)
	docker run --rm \
		-v ${ROOT_DIR}:/project \
		-w /project \
		-e GO111MODULE=on \
		golang:${GO_VERSION} \
		go mod init ${MOD_NAME}
endif

vendor: init
ifeq ($(FORCE_VENDOR), 1)
	docker run --rm \
		-v ${ROOT_DIR}:/project \
		-w /project \
		-e GO111MODULE=on \
		golang:${GO_VERSION} \
		go mod vendor
endif

test: vendor
	docker run --rm \
		-v ${ROOT_DIR}:/project \
		-w /project \
		-e GO111MODULE=on \
		golang:${GO_VERSION} \
		go test -mod vendor -v -race ./...

build: vendor
	rm -f ${APP_NAME} || true
	docker run --rm \
		-v ${ROOT_DIR}:/project \
		-w /project \
		golang:${GO_VERSION} \
		env GOOS=${GOOS} GOARCH=${GOARCH} CGO_ENABLED=${CGO_ENABLED} GO111MODULE=on go build -mod vendor -o ${APP_NAME} -v

image: build
	docker rmi ${DOCKER_IMAGE}:${DOCKER_TAG} || true
	docker build \
		--build-arg APP_NAME=${APP_NAME} \
		-f .docker/Dockerfile \
		-t ${DOCKER_IMAGE}:${DOCKER_TAG} \
		.

push: image
	docker login docker.io -u ${DOCKER_USER} -p ${DOCKER_PASSWORD}
	docker push ${DOCKER_IMAGE}:${DOCKER_TAG}
