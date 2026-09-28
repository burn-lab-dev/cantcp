#!/bin/sh
# Builds the cantcpd and cantcp-cli .deb packages for one architecture.
#
# Usage: scripts/build-deb.sh <version> <deb-arch> <go-arch> [goarm]
#
# The script is run from the repository root (the Makefile does this) and
# writes the packages into dist/.
set -eu

version=$1
debarch=$2
goarch=$3
goarm=${4:-}

root=$(pwd)
dist="$root/dist"
ldflags="-s -w -X github.com/burn-lab-dev/cantcp/internal/version.Version=v$version"

mkdir -p "$dist"
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT

# build_binary <cmd> <output>
build_binary() {
	cmd=$1
	out=$2
	echo "build $cmd for linux/$goarch$goarm (v$version)"
	if [ -n "$goarm" ]; then
		GOARM=$goarm CGO_ENABLED=0 GOOS=linux GOARCH=$goarch \
			go build -trimpath -ldflags "$ldflags" -o "$out" "./cmd/$cmd"
	else
		CGO_ENABLED=0 GOOS=linux GOARCH=$goarch \
			go build -trimpath -ldflags "$ldflags" -o "$out" "./cmd/$cmd"
	fi
}

# package <name> <deb-dir> <binary>
package() {
	name=$1
	debdir=$2
	binary=$3

	pkg="$stage/$name"
	mkdir -p "$pkg/usr/bin"

	cp -r "$root/deb/$debdir/." "$pkg/"
	build_binary "$name" "$pkg/usr/bin/$name"
	chmod 0755 "$pkg/usr/bin/$name"

	# Configuration examples shipped with the documentation.
	docdir="$pkg/usr/share/doc/$name/examples"
	mkdir -p "$docdir"
	if [ "$name" = "cantcpd" ]; then
		cp "$root"/examples/cantcpd-*.json "$docdir/"
	else
		cp "$root"/examples/cantcp-cli.json "$docdir/"
	fi

	# Compress the man pages: dpkg-deb stores files as they are.
	find "$pkg/usr/share/man" -type f ! -name '*.gz' -exec gzip -9 -n {} \; 2>/dev/null || true

	sed -i "s/@VERSION@/$version/; s/@ARCH@/$debarch/" "$pkg/DEBIAN/control"
	find "$pkg/DEBIAN" -type f -exec chmod 0755 {} \;
	chmod 0644 "$pkg/DEBIAN/control"
	[ -f "$pkg/DEBIAN/conffiles" ] && chmod 0644 "$pkg/DEBIAN/conffiles"

	dpkg-deb --build --root-owner-group "$pkg" "$dist/${name}_${version}_${debarch}.deb"
	echo "packaged dist/${name}_${version}_${debarch}.deb ($(du -h "$dist/${name}_${version}_${debarch}.deb" | cut -f1))"
}

package cantcpd cantcpd cantcpd
package cantcp-cli cantcp-cli cantcp-cli
