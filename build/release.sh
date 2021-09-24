#!/bin/bash
set -e

usage () {
    cat <<HELP_USAGE
Usage:

    $0 -v 
    $0 --dry-run -vv
    $0 --help 

   -h, --help         Prints this message 
   -d, --dry-run      Prints debugging info, but makes no changes 
   -v, --verbose      Prints more logs to the terminal
   -vv                Turns on verbose and prints each command out 
HELP_USAGE
    exit 1
}

verbose=0
function log () {
    if [[ $verbose -eq 1 ]]; then
        echo "$@"
    fi
}

function error () {
    RED=$(tput setaf 1)
    BOLD=$(tput bold)
    NC=$(tput sgr 0) 
    printf "${RED}${BOLD}[ERROR]:${NC}${BOLD} %s${NC}\n" "$@"
    if [[ $dry_run -eq 0 ]]; then
        exit 1
    fi
}

dry_run=0
while [[ $# -gt 0 ]]
do
key="$1"
case $key in
    -d|--dry-run)
    dry_run=1
    shift # past flag 
    ;;
    -v|--verborse)
    verbose=1
    shift # past flag 
    ;;
    -vv)
    set -x # log commands run as well
    verbose=1
    shift
    ;;
    -h|--help)
    usage
    exit 0
    ;;
    *)    # unknown option
    usage
    exit 1
    ;;
esac
done

if ! command -v git-sv &> /dev/null
then
    error "$(cat <<- ERR
git-sv could not be found, run the following to install:

    go install github.com/bvieira/sv4git/v2/cmd/git-sv@latest
ERR
)"
    exit 1
fi

if [[ $(git diff --staged --stat) != '' ]]; then
    error "There are currently files staged for commit. Either commit, or stash them before running again."
    

fi

if [[ $(git diff --stat) != '' ]]; then
    error "There are currently dirty files in the project. Either commit, or stash them before running again."
fi

current=v$(git-sv cv)
next=v$(git-sv nv)

printf "Current Version: %s\n" "${current}"
printf "Next Version: %s\n\n\n" "${next}"


if [[ $dry_run -eq 1 ]]; then
    git-sv cgl --add-next-version 
    exit 0
fi 

log "generating changelog..."
{
    git-sv cgl --add-next-version > CHANGELOG.md
} || {
    error "Error creating changelog!"
}
    
log "changelog created, adding to git..."
git add CHANGELOG.md
git commit -m -S "docs: Update CHANGELOG for $next release"

log "creating tag $next..."
git tag "$next"

cat <<RESULT
Changelog updated and committed!
Tag $next created!

CI will take it from here. Run the following to "send it":

    git push && git push --tags
RESULT
