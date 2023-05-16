#! /usr/bin/env bash

set -o errexit -o noclobber -o nounset -o pipefail

killall --exact --verbose 'bot' 'chrome' 'google-chrome' || true
