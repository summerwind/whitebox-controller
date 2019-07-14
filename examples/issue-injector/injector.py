#!/usr/bin/env python3

import sys
import json
import logging

def main():
    req = json.load(sys.stdin)
    body = json.loads(req["body"])

    res = {}
    if body["action"] == "opened":
        res["object"] = {
            "apiVersion": "whitebox.summerwind.dev/v1alpha1",
            "kind": "Issue",
            "metadata": {
                "name": "%s-%d" % (body["repository"]["name"], body["issue"]["number"]),
            },
            "spec": {
                "title": body["issue"]["title"],
                "url": body["issue"]["html_url"],
                "user": {
                    "login": body["issue"]["user"]["login"],
                    "url": body["issue"]["user"]["html_url"]
                },
                "repository": {
                    "name": body["repository"]["full_name"],
                    "url": body["repository"]["html_url"]
                }
            }
        }

    json.dump(res, sys.stdout)

if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        logging.exception("%s", e)
        sys.exit(1)
