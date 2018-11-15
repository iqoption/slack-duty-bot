#!/usr/bin/env bash

export ARTIFACTS_DIR=${ARTIFACTS_DIR:-"artifacts"}
mkdir -p ${ARTIFACTS_DIR}
rm -r ${ARTIFACTS_DIR}/*
for GOOS in darwin linux; do
    for GOARCH in 386 amd64; do
        make build GOOS=${GOOS} GOARCH=${GOARCH}
        tar -zcvf ${ARTIFACTS_DIR}/slack-duty-bot-${GOOS}-${GOARCH}.tar.gz slack-duty-bot
    done
done
