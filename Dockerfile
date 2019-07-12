FROM golang:1.12 as builder

ARG VERSION
ARG COMMIT

ENV GO111MODULE=on \
    GOPROXY=https://proxy.golang.org

RUN curl -L -o /tmp/download-binaries.sh https://raw.githubusercontent.com/kubernetes-sigs/testing_frameworks/master/integration/scripts/download-binaries.sh \
  && chmod +x /tmp/download-binaries.sh \
  && mkdir -p /usr/local/kubebuilder/bin \
  && /tmp/download-binaries.sh /usr/local/kubebuilder/bin

WORKDIR /go/src/github.com/summerwind/whitebox-controller
COPY go.mod go.sum .
RUN go mod download

COPY . /workspace
WORKDIR /workspace

RUN go vet ./...
RUN go test -v ./...
RUN CGO_ENABLED=0 go build -ldflags "-X main.VERSION=${VERSION} -X main.COMMIT=${COMMIT}" ./cmd/whitebox-controller
RUN CGO_ENABLED=0 go build -ldflags "-X main.VERSION=${VERSION} -X main.COMMIT=${COMMIT}" ./cmd/whitebox-gen

###################

FROM scratch

COPY --from=builder /workspace/whitebox-* /bin/

ENTRYPOINT ["/bin/whitebox-controller"]
