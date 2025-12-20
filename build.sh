#!/bin/sh
export CGO_ENABLED=1
export CC=x86_64-w64-mingw32-gcc
export GOOS=windows
export GOARCH=amd64
if [ ! -d "bin" ]; then
    mkdir bin
fi
if [ ! -f "rsrc.syso" ]; then
    rsrc -manifest ResourceInjector.manifest -ico favicon.ico -o rsrc.syso
fi
go build -ldflags="-H windowsgui -s -w" -o bin/ResourceInjector.exe
upx -9 bin/ResourceInjector.exe
