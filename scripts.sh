#!/usr/bin/env bash

cmd=$1

if [[ -z "$cmd" ]];then
  echo "Usage scripts.sh [tag|vendor]"
  exit 1
fi


tag() {
  vine="github.com/vine-io/vine"
  mods=$(find . -name "go.mod" | grep -v "vendor")

  for mod in $mods;do
    flag=$(cat "$mod" | grep $vine)
    echo "mod ${mod}"
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
}

vendor() {
  vine="github.com/vine-io/vine"
  mods=$(find . -name "go.mod" | grep -v "vendor")

  root=$PWD
  for mod in $mods;do
    version=$(cat ${mod} | grep -e "^go " | awk -F' ' '{print $2}')
    echo "mod ${mod} version=go:${version}"
    dir=$(dirname "$mod")
    cd "${dir:2}" && rm -fr vendor && rm -fr go.sum && go mod tidy -compat=${version} && go mod vendor
    cd "${root}"
  done
}

case $cmd in
tag)
  tag
  ;;
vendor)
  vendor
  ;;
*)
  echo "Usage scripts.sh [tag|vendor]"
  exit 1
  ;;
esac