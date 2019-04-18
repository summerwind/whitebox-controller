#!/usr/bin/env python3

import sys
import json
import logging

def main():
    state = json.load(sys.stdin)

    deps_len = len(state["dependents"]["deployment.v1.apps"])
    if deps_len > 1:
        logging.error("too many dependents")
        sys.exit(1)

    name = state["object"]["metadata"]["name"]
    namespace = state["object"]["metadata"]["namespace"]
    replicas = state["object"]["spec"]["replicas"]
    image = state["object"]["spec"]["image"]

    if deps_len == 0:
        state["dependents"]["deployment.v1.apps"] = [
            {
                "apiVersion": "apps/v1",
                "kind": "Deployment",
                "metadata": {
                    "name": name,
                    "namespace": namespace
                },
                "spec": {
                    "replicas": replicas,
                    "selector": {
                        "matchLabels": {
                            "containerset": name
                        }
                    },
                    "template": {
                        "metadata": {
                            "labels": {
                                "containerset": name
                            }
                        },
                        "spec": {
                            "containers": [
                                {
                                    "name": name,
                                    "image": image,
                                }
                            ]
                        }
                    }
                }
            }
        ]

        state["events"] = [
            {"type":"Normal", "reason":"CreateDeplpyment", "message":"deployment created"}
        ]
    else:
        deploy = state["dependents"]["deployment.v1.apps"][0]
        deploy["spec"]["replicas"] = replicas
        deploy["spec"]["template"]["spec"]["containers"][0]["image"] = image

        state["dependents"]["deployment.v1.apps"][0] = deploy

        availableReplicas = deploy.get("status", {}).get("availableReplicas", 0)
        state["object"]["status"] = {"healthyReplicas": availableReplicas}

    json.dump(state, sys.stdout)

if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        logging.exception("%s", e)
        sys.exit(1)
