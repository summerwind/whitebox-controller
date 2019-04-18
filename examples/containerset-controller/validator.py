#!/usr/bin/env python3

import sys
import json
import logging

def main():
    req = json.load(sys.stdin)

    allowed = True
    reason = ""

    replicas = req.get("object", {}).get("spec", {}).get("replicas")
    if replicas is None:
        allowed = False
        reason = "'spec.replicas' must be specified"
    elif replicas == 0:
        allowed = False
        reason = "'spec.replicas' must be non-zero value"

    image = req.get("object", {}).get("spec", {}).get("image")
    if image is None:
        allowed = False
        reason = "'spec.image' must be specified"
    elif image == "":
        allowed = False
        reason = "'spec.image' is empty"

    res = {
        "allowed": allowed,
        "status": {
            "reason": reason
        }
    }

    json.dump(res, sys.stdout)

if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        logging.exception("%s", e)
        sys.exit(1)
