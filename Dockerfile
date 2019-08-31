FROM golang:1.12 AS build

ENV GO111MODULE=on \
    GOPROXY=https://proxy.golang.org

RUN curl -L -o /tmp/download-binaries.sh https://raw.githubusercontent.com/kubernetes-sigs/testing_frameworks/master/integration/scripts/download-binaries.sh \
  && chmod +x /tmp/download-binaries.sh \
  && mkdir -p /usr/local/kubebuilder/bin \
  && /tmp/download-binaries.sh /usr/local/kubebuilder/bin

WORKDIR /go/src/github.com/summerwind/whitebox-controller
COPY go.mod go.sum ./
RUN go mod download

COPY . /workspace
WORKDIR /workspace

ARG VERSION
ARG COMMIT

RUN go vet ./...
RUN go test -v ./...
RUN CGO_ENABLED=0 go build -ldflags "-X main.VERSION=${VERSION} -X main.COMMIT=${COMMIT}" ./cmd/whitebox-controller
RUN CGO_ENABLED=0 go build -ldflags "-X main.VERSION=${VERSION} -X main.COMMIT=${COMMIT}" ./cmd/whitebox-gen

###################

FROM build AS release

ARG VERSION
ARG COMMIT

RUN mkdir release

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.VERSION=${VERSION} -X main.COMMIT=${COMMIT}" ./cmd/whitebox-controller \
  && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.VERSION=${VERSION} -X main.COMMIT=${COMMIT}" ./cmd/whitebox-gen \
  && tar zcf release/whitebox-controller-linux-amd64.tar.gz whitebox-controller whitebox-gen

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-X main.VERSION=${VERSION} -X main.COMMIT=${COMMIT}" ./cmd/whitebox-controller \
  && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-X main.VERSION=${VERSION} -X main.COMMIT=${COMMIT}" ./cmd/whitebox-gen \
  && tar zcf release/whitebox-controller-linux-arm64.tar.gz whitebox-controller whitebox-gen

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm go build -ldflags "-X main.VERSION=${VERSION} -X main.COMMIT=${COMMIT}" ./cmd/whitebox-controller \
  && CGO_ENABLED=0 GOOS=linux GOARCH=arm go build -ldflags "-X main.VERSION=${VERSION} -X main.COMMIT=${COMMIT}" ./cmd/whitebox-gen \
  && tar zcf release/whitebox-controller-linux-arm.tar.gz whitebox-controller whitebox-gen

RUN CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.VERSION=${VERSION} -X main.COMMIT=${COMMIT}" ./cmd/whitebox-controller \
  && CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.VERSION=${VERSION} -X main.COMMIT=${COMMIT}" ./cmd/whitebox-gen \
  && tar zcf release/whitebox-controller-darwin-amd64.tar.gz whitebox-controller whitebox-gen

###################

FROM scratch

COPY --from=build /workspace/whitebox-* /bin/

ENTRYPOINT ["/bin/whitebox-controller"]
