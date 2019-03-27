#!/bin/bash

set -e

if [ ! -e ca.pem ]; then
  echo "Generating certificate files for CA..."
  cfssl gencert -initca ca-csr.json | cfssljson -bare ca
fi

echo "Generating certificate files for server..."
cfssl gencert -ca=ca.pem -ca-key=ca-key.pem server-csr.json | cfssljson -bare server

echo "Generating caBundle..."
echo -n "caBundle: "
cat ca.pem | base64

echo "Generating secret..."
SERVER_PEM=`cat server.pem | base64`
SERVER_KEY_PEM=`cat server-key.pem | base64`

cat << EOL
---
apiVersion: v1
kind: Secret
metadata:
  name: noop-controller
  namespace: kube-system
type: Opaque
data:
  server.pem: ${SERVER_PEM}
  server-key.pem: ${SERVER_KEY_PEM}
EOL
