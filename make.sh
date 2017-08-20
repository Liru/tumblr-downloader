#!/bin/bash
NAME='tumblr-downloader'

for arch in amd64 386
do
	for os in linux darwin windows
	do
		echo "Building for $os $arch"
		env GOOS=$os GOARCH=$arch go build -ldflags "-X main.VERSION=$1"
		mkdir $NAME-$os-$arch
		EXECUTABLE="$NAME"
		if [ "$os" == "windows" ]; then
			EXECUTABLE="$EXECUTABLE.exe"
		fi
		mv $EXECUTABLE $NAME-$os-$arch
		cp LICENSE config.toml $NAME-$os-$arch/
		zip -9 -r $NAME-$os-$arch.zip $NAME-$os-$arch

		# remove all folders
		rm -r tumblr-downloader-$os-$arch/
	done
done
