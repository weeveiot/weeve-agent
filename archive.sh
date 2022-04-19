#!/usr/bin/env bash
image="$1"

echo "Create a tar archive"
tar -czC "./$image/" -f $image.tar.gz $(ls $image/) # avoid . being included in the tar
