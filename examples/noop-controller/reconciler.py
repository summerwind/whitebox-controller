#!/usr/bin/env python3

import sys
import json
import logging

def main():
    state = json.load(sys.stdin)

    phase = state.get("resource", {}).get("status", {}).get("phase", "")
    if phase == "":
        state["resource"]["status"] = {"phase":"completed"}
        state["events"] = [
            {"type":"Normal", "reason":"Completed", "message":"Done"}
        ]

    json.dump(state, sys.stdout)

if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        logging.exception("%s", e)
        sys.exit(1)
