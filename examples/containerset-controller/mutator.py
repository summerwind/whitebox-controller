#!/usr/bin/env python3

import sys
import json
import base64
import logging

def main():
    req = json.load(sys.stdin)

    replace = False
    patch = {}

    replicas = req.get("object", {}).get("spec", {}).get("replicas")
    if replicas is None:
        patch = {"op": "add", "path": "/spec/replicas", "value": 0}
    elif replicas == 0:
        patch = {"op": "replace", "path": "/spec/replicas", "value": 1}

    res = {"allowed": True}
    if len(patch) != 0:
        res["patchType"] = "JSONPatch"
        res["patch"] = base64.b64encode(json.dumps([patch]).encode('utf-8')).decode()

    json.dump(res, sys.stdout)

if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        logging.exception("%s", e)
        sys.exit(1)
