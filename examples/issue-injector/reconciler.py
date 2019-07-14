#!/usr/bin/env python3

import sys
import json
import logging

def main():
    state = json.load(sys.stdin)

    phase = state.get("object", {}).get("status", {}).get("phase", "")
    if phase == "":
        state["object"]["status"] = {"phase": "created"}

    json.dump(state, sys.stdout)

if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        logging.exception("%s", e)
        sys.exit(1)
