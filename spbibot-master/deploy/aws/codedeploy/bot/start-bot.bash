#! /usr/bin/env bash

set -o errexit -o noclobber -o nounset -o pipefail

cd '/opt/spbi-bot'
'./bot' &>'/dev/null' &
