#!/bin/sh
# NOTE: This is meant to be run inside the repo root as ./tools/setup-git-hooks.sh

dir=$(pwd)
ln -sf $dir/tools/format-commit-changes.sh $dir/.git/hooks/pre-commit