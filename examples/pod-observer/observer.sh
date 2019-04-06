#!/bin/bash

# Read current state from stdio.
STATE=`cat -`

# Get current phase of the Pod.
POD_NAME=`echo "${STATE}" | jq -r '.metadata.name'`
POD_NAMESPACE=`echo "${STATE}" | jq -r '.metadata.namespace'`
POD_PHASE=`echo "${STATE}" | jq -r '.status.phase'`

# Generate message
MESSAGE="Pod ${POD_NAMESPACE}/${POD_NAME} is ${POD_PHASE}"

# Write message to the log
echo "${MESSAGE}" >> pod.log

# Write message to stder
echo "${MESSAGE}" >&2
