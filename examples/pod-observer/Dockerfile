FROM summerwind/whitebox-controller:latest AS base

#######################################

FROM ubuntu:18.04

RUN apt update \
  && apt install -y jq \
  && rm -rf /var/lib/apt/lists/\*

COPY --from=base /bin/whitebox-controller /bin/whitebox-controller

COPY observer.sh /observer.sh
COPY config.yaml /config.yaml

ENTRYPOINT ["/bin/whitebox-controller"]
