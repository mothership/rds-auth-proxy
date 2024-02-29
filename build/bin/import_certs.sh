#!/bin/bash
set -e

mydir=/tmp/rds-ca
if [ ! -e "${mydir}" ]; then
    mkdir -p "${mydir}"
fi

pushd "${mydir}"
curl -sS "https://truststore.pki.rds.amazonaws.com/global/global-bundle.pem" >${mydir}/rds-combined-ca-bundle.pem
awk 'split_after == 1 {n++;split_after=0} /-----END CERTIFICATE-----/ {split_after=1}{print > "rds-ca-" n ""}' <${mydir}/rds-combined-ca-bundle.pem

for CERT in rds-ca-*; do
    mv "$CERT" "/usr/local/share/ca-certificates/aws-rds-ca-$(basename $CERT).crt"
done

popd
rm -rf ${mydir}
update-ca-certificates
