#!/bin/bash 

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"
PROJECT_DIR="${SCRIPT_DIR}/.."

OUT_DIR="${PROJECT_DIR}/dist"

cd $PROJECT_DIR

rm -rf $OUT_DIR
mkdir $OUT_DIR
go mod download && go mod verify
go build -v -o "$OUT_DIR/app" .


mkdir -p "$OUT_DIR/migrate/"
mkdir -p "$OUT_DIR/scripts/"
cp -r ./migrate/*.sql "$OUT_DIR/migrate/"
cp -r ./scripts "$OUT_DIR/scripts/"

echo "Done in $OUT_DIR"



