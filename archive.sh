image="$1"

echo "Create a tar archive"
tar -cC './${image}/' $(ls -1 ${image}/) -f ${image}.tar # avoid . being included in the tar
gzip ${image}.tar