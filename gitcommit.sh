#!/bin/bash
#version=$(if [ "$(git describe --tags --abbrev=0 2>/dev/null)" != "" ]; then git describe --tags --abbrev=0; else git log --pretty=format:'%h' -n 1; fi)
#version=$(git tag -l "v*" --sort=-creatordate | head -n 1)
version=$(git tag -l "[0-99]*.[0-99]*.[0-99]*" --sort=-creatordate | head -n 1)
#git tag --sort=-creatordate | head -n 1
#git tag -l "v*" --sort=-creatordate | head -n 1
#git tag -l "v[0-99][0-99].[0-99][0-99].[0-99][0-99]" --sort=-v:refname | head -n 1
#git tag -l "v*.*.*" --sort=-v:refname | head -n 1
# git tag -l "[0-99]*.[0-99]*.[0-99]*" --sort=-creatordate | head -n 1
function upgradeVersion() {
  if [ "$version" = "" ]; then
    version="0.0.0"
  else
    v3=$(echo $version | awk -F'.' '{print($3);}')
    v2=$(echo $version | awk -F'.' '{print($2);}')
    v1=$(echo $version | awk -F'.' '{print($1);}')
    if [[ $(expr $v3 \>= 99) == 1 ]]; then
      v3=0
      if [[ $(expr $v2 \>= 99) == 1 ]]; then
        v2=0
        v1=$(expr $v1 + 1)
      else
        v2=$(expr $v2 + 1)
      fi
    else
      v3=$(expr $v3 + 1)
    fi
    version="$v1.$v2.$v3"
  fi
}

function todir() {
  pwd
}

function pull() {
  todir
  echo "git pull"
  git pull
}

function forcepull() {
  todir
  echo "git fetch --all && git reset --hard origin/master && git pull"
  git fetch --all && git reset --hard origin/$1 && git pull
}

function tag() {
  echo "===>${version}"
  git add .
  git commit -m "release ${version}"
  git tag -a $version -m "release ${version}"
  git push origin $version
}

function push() {
  commit=""
  if [ ! -n "$1" ]; then
    commit="$(date '+%Y-%m-%d %H:%M:%S') by ${USER}"
  else
    commit="$1 by ${USER}"
  fi
  echo $commit
  git add .
  git commit -m "${version} $commit"
  #  git push -u origin main
  echo "提交代码"
  git push
  echo "打tag标签"
  tag
}

function main_pre() {
  #1. 更新版本号
  upgradeVersion
}

function utag() {
    echo "请输入分支名称："
    read tag
    forcepull $tag
}

function tagAndGitPush() {
    echo "请输入标签提交commit:"
    read commit
    commit="$commit $(date '+%Y-%m-%d %H:%M:%S') by ${USER}"
    vtag="$(date '+%Y.%m.%d.%H.%M.%S')"
    git add .
    git commit -m "${commit}"
    git tag -a v$vtag -m "${commit}"
    git push origin v$vtag
}

function forceupdate() {
    echo "1. master"
    echo "2. main"
    echo "3. 输入分支"
    echo "请输入编号:"
    read index

    case "$index" in
    [1]) (forcepull master);;
    [2]) (pull main);;
    [3]) (utag);;
    *) echo "exit" ;;
  esac
}

function m() {
    echo "1. 强制更新"
    echo "2. 普通更新"
    echo "3. 提交项目"
    echo "4. 打tag标签"
    echo "请输入编号:"
    read index

    case "$index" in
    [1]) (forceupdate);;
    [2]) (pull);;
    [3]) (push);;
    [4]) (tagAndGitPush);;
    *) echo "exit" ;;
  esac
}

function main() {
  main_pre
    case $1 in
    pull) (pull) ;;
       m) (m) ;;
      -f) (forcepull) ;;
       *) (push $1)  ;;
    esac
}

function test() {
    echo "====>>>>${version}"
    upgradeVersion
    echo "====>>>>${version}"
}

main m
#upgradeVersion
#test
