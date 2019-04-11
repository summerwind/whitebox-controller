FROM golang:1.12.1 as base

ENV GO111MODULE=on

WORKDIR /go/src/github.com/summerwind/whitebox-controller
COPY go.mod go.sum .
RUN go mod download

###################

FROM base as builder

ARG BUILD_ARG

COPY . /workspace
WORKDIR /workspace

RUN go vet ./... && go test -v ./...
RUN CGO_ENABLED=0 go build ${BUILD_FLAGS} ./cmd/whitebox-controller
RUN CGO_ENABLED=0 go build ${BUILD_FLAGS} ./cmd/whitebox-gen

###################

FROM scratch

COPY --from=builder /workspace/whitebox-* /bin/

ENTRYPOINT ["/bin/whitebox-controller"]
