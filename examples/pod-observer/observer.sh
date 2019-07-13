#!/bin/bash

# Read current state from stdio.
STATE=`cat -`

# Get current phase of the Pod.
POD_NAME=`echo "${STATE}" | jq -r '.object.metadata.name'`
POD_NAMESPACE=`echo "${STATE}" | jq -r '.object.metadata.namespace'`
POD_PHASE=`echo "${STATE}" | jq -r '.object.status.phase'`

# Generate message
MESSAGE="Pod ${POD_NAMESPACE}/${POD_NAME} is ${POD_PHASE}"

# Write message to the log
NOW=`date +%s`
echo "${NOW}: ${MESSAGE}" >> pod.log

# Write message to stder
echo "${MESSAGE}" >&2
