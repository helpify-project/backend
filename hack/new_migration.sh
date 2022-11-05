#!/usr/bin/env bash
set -euo pipefail

name="${1?must specify migration name}"

rootdir="$(git rev-parse --show-toplevel)"
sqldir="${rootdir}/internal/database/migrations"

mkdir -p "${sqldir}"
go run github.com/pressly/goose/v3/cmd/goose -dir "${sqldir}" create "${name}" sql
