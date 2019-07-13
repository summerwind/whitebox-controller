#!/bin/bash

# Read current state from stdio.
STATE=`cat -`

# Read phase from object.
PHASE=`echo "${STATE}" | jq -r '.object.status.phase'`

# Reconcile object.
if [ "${PHASE}" != "completed" ]; then
  # Write message to stder.
  NOW=`date "+%Y/%m/%d %H:%M:%S"`
  echo -n "${NOW} message: " >&2
  echo "${STATE}" | jq -r '.object.spec.message' >&2

  # Set `.status.phase` field to the resource.
  STATE=`echo "${STATE}" | jq -r '.object.status.phase = "completed"'`
fi

# Write new state to stdio.
echo "${STATE}"
