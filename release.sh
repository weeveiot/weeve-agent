#!/bin/bash

create_debs(){
    for ARCH in amd64 arm arm64
    do
        mkdir -p deb-${ARCH}
        cp -r DEBIAN deb-${ARCH}

        # populate control file
        echo "Package: weeve-agent
Version: ${VERSION}
License: GPL-3.0
Architecture: ${ARCH}
Maintainer: ${FULL_NAME} <${EMAIL}>
Section: misc
Priority: optional
Homepage: https://weeve.network
Description: A client to manage docker containers on IoT devices." > deb-${ARCH}/DEBIAN/control

        mkdir -p deb-${ARCH}/var/lib/weeve-agent
        cp ca.crt deb-${ARCH}/var/lib/weeve-agent

        mkdir -p deb-${ARCH}/lib/systemd/system/
        cp weeve-agent.service deb-${ARCH}/lib/systemd/system

        mkdir -p deb-${ARCH}/usr/bin/
        cp bin/weeve-agent-linux-${ARCH} deb-${ARCH}/usr/bin/weeve-agent

        # TODO: generate the changelog

        dpkg-deb --build deb-${ARCH} "weeve-agent_${VERSION}_${ARCH}.deb"
    done
}

create_sign_release(){
    cp weeve.gpg apt-repo
    cd apt-repo
    mkdir -p pool/main/

    for ARCH in amd64 arm arm64
    do
        # create weeve-agent.list file
        echo "deb [arch=${ARCH} signed-by=/etc/apt/trusted.gpg.d/weeve.gpg] http://${BUCKET}.s3.amazonaws.com stable main" > weeve-agent-${ARCH}.list
        # copy the deb
        cp "../weeve-agent_${VERSION}_${ARCH}.deb" pool/main/

        # create Packages* files
        mkdir -p dists/stable/main/binary-${ARCH}
        dpkg-scanpackages --arch ${ARCH} --multiversion pool/ > dists/stable/main/binary-${ARCH}/Packages
        cat dists/stable/main/binary-${ARCH}/Packages | gzip -9 > dists/stable/main/binary-${ARCH}/Packages.gz
    done

    # create *Release* files
    cd dists/stable
    apt-ftparchive release . > Release
    gpg -abs -o - Release > Release.gpg
    gpg --clearsign -o - Release > InRelease
    cd ../../..
}

configure_gpg(){
  echo -n "${GPG_SIGNING_KEY}" | base64 --decode | gpg --import
}

FULL_NAME="Paul Gaiduk"
EMAIL=paul.gaiduk@weeve.network

VERSION=$(git tag | sort -V | tail -n 1)
VERSION=${VERSION#v}

BUCKET=weeve-agent-ppa

"$@"
