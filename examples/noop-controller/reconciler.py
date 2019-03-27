#!/usr/bin/env python3

import sys
import json
import logging

def main():
    state = json.load(sys.stdin)

    if "resource" in state:
        state["resource"]["status"] = {"phase":"completed"}

    json.dump(state, sys.stdout)

if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        logging.exception("%s", e)
        sys.exit(1)
