#!/usr/bin/env bash

set -euxo pipefail

mkdir -p /tmp/mcm-unpacker-download

wget https://github.com/STBoyden/newvegas-linux-mcm-extractor/releases/download/release/newvegas-linux-mcm-extractor
chmod +x newvegas-linux-mcm-extractor
mv newvegas-linux-mcm-extractor /tmp/mcm-unpacker-download/

/tmp/mcm-unpacker-download/newvegas-linux-mcm-extractor

rm /tmp/mcm-unpacker-download/newvegas-linux-mcm-extractor
