#!/bin/sh
# SPDX-License-Identifier: MIT
set -eu

namespace="open-idb"
secret_name="idbridge-tls"
tls_dir=".local/tls"
ca_key="$tls_dir/dev-idbridge-ca.key"
ca_crt="$tls_dir/dev-idbridge-ca.crt"
leaf_key="$tls_dir/local.test.key"
leaf_csr="$tls_dir/local.test.csr"
leaf_crt="$tls_dir/local.test.crt"
leaf_conf="$tls_dir/local.test.openssl.cnf"

mkdir -p "$tls_dir"
chmod 700 "$tls_dir"

if [ ! -f "$ca_key" ] || [ ! -f "$ca_crt" ]; then
  openssl genrsa -out "$ca_key" 4096
  openssl req -x509 -new -nodes -key "$ca_key" -sha256 -days 3650 \
    -subj "/CN=IdBridge Local Dev Root CA" \
    -out "$ca_crt"
  chmod 600 "$ca_key"
fi

cat > "$leaf_conf" <<'EOF'
[req]
default_bits = 2048
prompt = no
default_md = sha256
distinguished_name = dn
req_extensions = v3_req

[dn]
CN = *.local.test

[v3_req]
basicConstraints = CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = local.test
DNS.2 = *.local.test
DNS.3 = idbridge.local.test
EOF

openssl genrsa -out "$leaf_key" 2048
openssl req -new -key "$leaf_key" -out "$leaf_csr" -config "$leaf_conf"
openssl x509 -req -in "$leaf_csr" -CA "$ca_crt" -CAkey "$ca_key" -CAcreateserial \
  -out "$leaf_crt" -days 825 -sha256 -extensions v3_req -extfile "$leaf_conf"
chmod 600 "$leaf_key"

if [ "${TRUST_LOCAL_CA:-0}" = "1" ] && command -v security >/dev/null 2>&1; then
  security add-trusted-cert -d -r trustRoot \
    -k "$HOME/Library/Keychains/login.keychain-db" "$ca_crt"
fi

kubectl create namespace "$namespace" --dry-run=client -o yaml | kubectl apply -f -
kubectl -n "$namespace" create secret tls "$secret_name" \
  --cert="$leaf_crt" \
  --key="$leaf_key" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl apply -f deploy/k8s/orbstack/ingress.yaml

if [ "${TRUST_LOCAL_CA:-0}" = "1" ]; then
  echo "Local TLS is trusted for https://idbridge.local.test"
else
  echo "Local TLS secret is installed for https://idbridge.local.test"
  echo "To remove browser warnings on macOS, rerun: TRUST_LOCAL_CA=1 make k8s-local-tls"
fi
