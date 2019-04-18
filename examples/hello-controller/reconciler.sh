#!/bin/bash

# Read current state from stdio.
STATE=`cat -`

# Write message to stder
echo "${STATE}" | jq -r '.object.spec.message' >&2

# Set `.status.phase` field to the resource
NEW_STATE=`echo "${STATE}" | jq -r '.object.status.phase = "completed"'`

# Write new state to stdio.
echo "${NEW_STATE}"
