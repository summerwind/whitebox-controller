FROM summerwind/whitebox-controller:latest AS base

#######################################

FROM ubuntu:18.04

RUN apt update \
  && apt install -y python3 \
  && rm -rf /var/lib/apt/lists/*

COPY --from=base /bin/whitebox-controller /bin/whitebox-controller

COPY reconciler.py /reconciler.py
COPY injector.py   /injector.py
COPY config.yaml   /config.yaml

ENTRYPOINT ["/bin/whitebox-controller"]
