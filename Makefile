BIN_DIR=_output/cmd/bin
REPO_PATH="k8s.io/k8s-device-plugin"
REL_OSARCH="linux/amd64"
GitSHA=`git rev-parse HEAD`
Date=`date "+%Y-%m-%d %H:%M:%S"`
RELEASE_VERSION="v0.1.0"
IMG_BUILDER=docker
LD_FLAGS=" \
    -X '${REPO_PATH}/version.GitSHA=${GitSHA}' \
    -X '${REPO_PATH}/version.Built=${Date}'   \
    -X '${REPO_PATH}/version.Version=${RELEASE_VER}'"

build: all

all: init k8s-device-plugin

k8s-device-plugin:
	GOOS=linux GOARCH=amd64 go build -o ${BIN_DIR}/k8s-device-plugin .

init:
	mkdir -p ${BIN_DIR}

clean:
	rm -fr ${BIN_DIR}

images:
	@echo "version: ${RELEASE_VERSION}"
	${IMG_BUILDER} build -t xx.oa.com/kubeflow/k8s-device-plugin:${RELEASE_VERSION} .

.PHONY: clean
