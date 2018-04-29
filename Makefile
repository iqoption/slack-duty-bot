all: build

APP?=slack-duty-bot
ROOT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
BUILD_OS?=linux
BUILD_ARCH?=amd64
DOCKER_IMAGE?=insidieux/${APP}
DOCKER_TAG?=1.0.0
DOCKER_USER?=user
DOCKER_PASSWORD?=password
SDB_SLACK_TOKEN?=some-token
SDB_SLACK_KEYWORD?=keyword

clean:
	rm -f ${APP}

dep-install:
	go get -v -u github.com/golang/dep/cmd/dep

dep-ensure: dep-install
	rm -rf vendor
	dep ensure

test: dep-ensure
	go test -v -race ./...

build: clean dep-ensure
	env GOOS=${BUILD_OS} GOARCH=${BUILD_ARCH} CGO_ENABLED=0 go build -v -o ${APP}

container: build
	docker rmi ${DOCKER_IMAGE}:${DOCKER_TAG} || true
	docker build --build-arg APP=${APP} -f .docker/Dockerfile -t ${DOCKER_IMAGE}:${DOCKER_TAG} .

run: container
	docker stop ${APP} || true && docker rm ${APP} || true
	docker run --name ${APP} --rm \
		-e SDB_SLACK_TOKEN=${SDB_SLACK_TOKEN} \
		${DOCKER_IMAGE}:${DOCKER_TAG} \
		--slack.keyword ${SDB_SLACK_KEYWORD}

push: container
	docker login docker.io -u ${DOCKER_USER} -p ${DOCKER_PASSWORD}
	docker push ${DOCKER_IMAGE}:${DOCKER_TAG}
