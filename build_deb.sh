#!/bin/bash
set -e

SCRIPT_BASE_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"
BUILD_DIR=${1%/}

if [ -f "${BUILD_DIR}" ]; then
	echo "Cannot build into '${BUILD_DIR}': it is a file."
	exit 1
fi

if [ -d "${BUILD_DIR}" ]; then
  rm -rfv ${BUILD_DIR}/*
fi


mkdir -pv ${BUILD_DIR}/src/github.com/twitchyliquid64/subnet
cp -rv DEBIAN/ subnet/ vendor/ *.go ${BUILD_DIR}/src/github.com/twitchyliquid64/subnet
export GOPATH="${BUILD_DIR}"

mkdir -pv "${BUILD_DIR}/usr/bin"
go build -v -o "${BUILD_DIR}/usr/bin/subnet" github.com/twitchyliquid64/subnet
rm -rf "${BUILD_DIR}/src"


mkdir -pv "${BUILD_DIR}/DEBIAN"
cp -rv ${SCRIPT_BASE_DIR}/DEBIAN/* "${BUILD_DIR}/DEBIAN"

ARCH=`dpkg --print-architecture`
sed -i "s/ARCH/${ARCH}/g" "${BUILD_DIR}/DEBIAN/control"


cat > ${BUILD_DIR}/usr/bin/subnet-make-certs << "EOF"
#!/bin/bash
set -e

subnet --mode init-server-certs --cert server.certPEM --key server.keyPEM --ca ca.certPEM --ca_key ca.keyPEM && \
echo "" && \
echo "Wrote: server.certPEM, server.keyPEM, ca.certPEM, ca.keyPEM." && \
echo "Keep them safe."
EOF
chmod +x ${BUILD_DIR}/usr/bin/subnet-make-certs


dpkg-deb --build "${BUILD_DIR}" ./
