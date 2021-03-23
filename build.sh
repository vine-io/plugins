#!/usr/bin/env bash

vine="github.com/lack-io/vine"
mods=$(find . -name "go.mod" | grep -v "vendor")

for mod in $mods;do
  flag=$(cat "$mod" | grep $vine)
  if [[ -n $flag ]];then
    v=$(echo "${flag}" | awk -F' ' '{print $NF}')
    dir=$(dirname "$mod")
    version=${dir:2}/$v
    vv=$(git tag --list | grep "$version")
    if [[ -z $vv ]];then
      git add .
      git commit -m "$version"
      git tag "$version"
      echo "add tag $version"
    fi
  fi
done