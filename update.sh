#!/bin/sh

JSON=$(curl -sL https://api.github.com/repos/mehtaarn000/SuperSourceControl/releases)
SSCPATH=$(which ssc)

ZIPBALL=$(echo "$JSON" | tr '\r\n' ' ' | jq -r '.[0] | .zipball_url')
curl -L "$ZIPBALL" > /tmp/ssc.zip
unzip /tmp/ssc.zip -d /tmp/SuperSourceControl

ZIPDIR=$(ls /tmp/SuperSourceControl)
FULLPATH="/tmp/SuperSourceControl/$ZIPDIR/ssc.go"
go build -o "/tmp/SuperSourceControl/$ZIPDIR/ssc" "$FULLPATH"
mv "/tmp/SuperSourceControl/$ZIPDIR/ssc" "$SSCPATH"
rm -rf "/tmp/SuperSourceControl"