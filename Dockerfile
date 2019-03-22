FROM golang:1.12.1 as builder

ENV GO111MODULE=on

COPY . /workspace
WORKDIR /workspace

RUN CGO_ENABLED=0 go build .

###################

FROM scratch

COPY --from=builder /workspace/whitebox-controller /bin/whitebox-controller

ENTRYPOINT ["/bin/whitebox-controller"]
