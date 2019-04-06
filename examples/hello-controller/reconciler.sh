#!/bin/bash

# Read current state from stdio.
STATE=`cat -`

# Write message to stder
echo "${STATE}" | jq -r '.resource.spec.message' >&2

# Set `.status.phase` field to the resource
NEW_STATE=`echo "${STATE}" | jq -r '.resource.status.phase = "completed"'`

# Write new state to stdio.
echo "${NEW_STATE}"
