FROM golang:1.12.1 as builder

ARG BUILD_ARG

ENV GO111MODULE=on

COPY . /workspace
WORKDIR /workspace

RUN CGO_ENABLED=0 go build ${BUILD_FLAGS} .

###################

FROM scratch

COPY --from=builder /workspace/whitebox-controller /bin/whitebox-controller

ENTRYPOINT ["/bin/whitebox-controller"]
