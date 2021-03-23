#!/usr/bin/env bash

vine="github.com/lack-io/vine"
mods=`find . -name "go.mod" | grep -v "vendor"`

for mod in $mods;do
  flag=`cat $mod | grep $vine`
  if [[ -n $flag ]];then
    v=`echo $flag | awk -F' ' '{print $NF}'`
    vv=`git tag --list | grep $v`
    if [[ -z $vv ]];then
      dir=`dirname $mod`
      git add $dir
      version=${dir:2}/$v
      git commit -m "$version"
      echo $version
#      git tag $version
    fi
  fi
done