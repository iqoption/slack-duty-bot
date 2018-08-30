all: build

# strict variables
APP:=slack-duty-bot
ROOT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

# build go binary variables
SRC_DIR?=$(shell echo '/go/$(shell echo ${ROOT_DIR} | awk -F'/go/' '{print $$2}')')
BUILD_OS?=linux
BUILD_ARCH?=amd64

# docker build variables
DOCKER_IMAGE?=insidieux/${APP}
DOCKER_TAG?=1.0.0
DOCKER_USER?=user
DOCKER_PASSWORD?=password

# run variables
SDB_SLACK_TOKEN?=some-token
SDB_SLACK_KEYWORD?=keyword

dep-ensure:
	rm -r vendor || true
	rm -r .vendor-new || true
	docker run --rm \
		-v ${ROOT_DIR}:${SRC_DIR} \
		-w ${SRC_DIR} \
		golang:1.10 \
		bash -c "go get -v -u github.com/golang/dep/cmd/dep && dep ensure"

test: dep-ensure
	docker run --rm \
		-v ${ROOT_DIR}:${SRC_DIR} \
		-w ${SRC_DIR} \
		golang:1.10 \
		go test -v -race ./...

build: dep-ensure
	rm ${APP} || true
	docker run --rm \
		-v ${ROOT_DIR}:${SRC_DIR} \
		-w ${SRC_DIR} \
		golang:1.10 \
		env GOOS=${BUILD_OS} GOARCH=${BUILD_ARCH} CGO_ENABLED=0 go build -o ${APP} -v main.go

image: build
	docker rmi ${DOCKER_IMAGE}:${DOCKER_TAG} || true
	docker build \
		--build-arg APP=${APP} \
		-f .docker/Dockerfile \
		-t ${DOCKER_IMAGE}:${DOCKER_TAG} \
		.

run: image
	docker stop ${APP} || true
	docker rm ${APP} || true
	docker run \
		--name ${APP} \
		--rm \
		-e SDB_SLACK_TOKEN=${SDB_SLACK_TOKEN} \
		${DOCKER_IMAGE}:${DOCKER_TAG} \
		--slack.keyword ${SDB_SLACK_KEYWORD}

push: image
	docker login docker.io -u ${DOCKER_USER} -p ${DOCKER_PASSWORD}
	docker push ${DOCKER_IMAGE}:${DOCKER_TAG}
