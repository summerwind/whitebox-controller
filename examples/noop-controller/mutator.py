#!/usr/bin/env python3

import sys
import json
import base64
import logging

ANNOTATION_KEY = "noop"

def main():
    req = json.load(sys.stdin)

    replace = False
    patch = {}

    annotations = req.get("object", {}).get("metadata", {}).get("annotations", {})
    if ANNOTATION_KEY in annotations:
        replace = True

    if replace:
        patch = {"op": "replace", "path": ("/metadata/annotations/%s" % ANNOTATION_KEY), "value": "true"}
    else:
        patch = {"op": "add", "path": "/metadata/annotations", "value": {ANNOTATION_KEY: "true"}}

    res = {
        "allowed": True,
        "patchType": "JSONPatch",
        "patch": base64.b64encode(json.dumps([patch]).encode('utf-8')).decode(),
    }

    json.dump(res, sys.stdout)

if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        logging.exception("%s", e)
        sys.exit(1)
