FROM golang:1.12 as builder

ARG BUILD_ARG
ENV GO111MODULE=on \
    GOPROXY=https://proxy.golang.org

WORKDIR /go/src/github.com/summerwind/whitebox-controller
COPY go.mod go.sum .
RUN go mod download

COPY . /workspace
WORKDIR /workspace

RUN go vet ./...
RUN go test -v ./...
RUN CGO_ENABLED=0 go build ${BUILD_FLAGS} ./cmd/whitebox-controller
RUN CGO_ENABLED=0 go build ${BUILD_FLAGS} ./cmd/whitebox-gen

###################

FROM scratch

COPY --from=builder /workspace/whitebox-* /bin/

ENTRYPOINT ["/bin/whitebox-controller"]
