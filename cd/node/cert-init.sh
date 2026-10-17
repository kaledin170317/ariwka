#!/bin/sh
set -eu

if [ ! -s /certs/tls.crt ] || [ ! -s /certs/tls.key ]; then
  openssl req -x509 -newkey rsa:2048 -nodes \
    -keyout /certs/tls.key -out /certs/tls.crt \
    -days 3650 -subj /CN=vpn
fi

chown 65532:65532 /certs/tls.crt /certs/tls.key
chmod 644 /certs/tls.crt
chmod 600 /certs/tls.key

# Keep older control binaries working during the transition.
ln -sf tls.crt /certs/hy2_cert.pem
ln -sf tls.key /certs/hy2_key.pem
