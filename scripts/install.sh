#!/usr/bin/env bash

set -euxo pipefail

mkdir -p /tmp/mcm-unpacker-download

wget https://github.com/STBoyden/newvegas-linux-mcm-extractor/releases/download/release/newvegas-linux-mcm-extractor
chmod +x newvegas-linux-mcm-extractor

./newvegas-linux-extractor