#!/usr/bin/env python3

import sys
import json
import logging

def main():
    req = json.load(sys.stdin)

    allowed = True
    reason = ""

    if "spec" in req["object"]:
        data = req["object"]["spec"].get("data")
        if data == "":
            allowed = False
            reason = "'spec.data' is empty"
    else:
        allowed = False
        reason = "'spec' is empty"

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
