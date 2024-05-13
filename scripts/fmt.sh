#!/usr/bin/env bash

if [[ ! $(command -v just) ]]; then
  echo "just not found. Please install."
  exit 1
fi

just fmt