#!/bin/sh
# Builds a static APT repository from .deb packages.
#
# Usage: scripts/apt-repo.sh <debs-dir> <repo-dir> [suite]
#
# The repository is laid out as:
#
#   pool/main/<first-letter>/<package>/*.deb
#   dists/<suite>/main/binary-<arch>/Packages[.gz]
#   dists/<suite>/Release[.gpg], dists/<suite>/InRelease
#   cantcp.gpg
#
# When the APT_GPG_KEY environment variable holds an ASCII-armored private
# key, the Release file is signed and the public key is exported to
# cantcp.gpg. Without the variable the repository is built unsigned (used by
# the local smoke test only).
#
# Requires dpkg-scanpackages and apt-ftparchive (dpkg-dev, apt-utils).
set -eu

debs=$1
repo=$2
suite=${3:-stable}

if [ ! -d "$debs" ]; then
	echo "apt-repo: $debs: no such directory" >&2
	exit 1
fi

rm -rf "$repo"
mkdir -p "$repo/dists/$suite/main"
# Make the path absolute: the script changes into dists/<suite> before it
# writes the signature files back into the repository root.
repo=$(cd "$repo" && pwd)

arches=""
for deb in "$debs"/*.deb; do
	[ -e "$deb" ] || continue
	pkg=$(dpkg-deb -f "$deb" Package)
	arch=$(dpkg-deb -f "$deb" Architecture)
	mkdir -p "$repo/pool/main/${pkg%"${pkg#?}"}/$pkg"
	cp "$deb" "$repo/pool/main/${pkg%"${pkg#?}"}/$pkg/"
	case " $arches " in
	*" $arch "*) ;;
	*) arches="$arches $arch" ;;
	esac
done
arches=$(echo "$arches" | sed 's/^ //')

if [ -z "$arches" ]; then
	echo "apt-repo: no .deb files in $debs" >&2
	exit 1
fi

for arch in $arches; do
	dir="$repo/dists/$suite/main/binary-$arch"
	mkdir -p "$dir"
	(cd "$repo" && dpkg-scanpackages --arch "$arch" pool /dev/null > "dists/$suite/main/binary-$arch/Packages")
	gzip -9 -n "$dir/Packages"
done

cd "$repo/dists/$suite"
apt-ftparchive \
	-o APT::FTPArchive::Release::Origin="cantcp" \
	-o APT::FTPArchive::Release::Label="cantcp" \
	-o APT::FTPArchive::Release::Suite="$suite" \
	-o APT::FTPArchive::Release::Codename="$suite" \
	-o APT::FTPArchive::Release::Architectures="$arches" \
	-o APT::FTPArchive::Release::Components="main" \
	release . > Release

if [ -n "${APT_GPG_KEY:-}" ]; then
	echo "$APT_GPG_KEY" | gpg --batch --import
	key=$(gpg --batch --list-secret-keys --with-colons | awk -F: '/^sec:/ {print $5; exit}')
	gpg --batch --yes --local-user "$key" --armor --detach-sign -o Release.gpg Release
	gpg --batch --yes --local-user "$key" --clearsign -o InRelease Release
	gpg --batch --yes --armor --export "$key" > "$repo/cantcp.gpg"
	echo "apt-repo: Release signed with $key"
else
	echo "apt-repo: APT_GPG_KEY is not set, the repository is not signed" >&2
fi

echo "apt-repo: $repo built for arches:$arches"
