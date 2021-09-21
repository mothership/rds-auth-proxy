#!/bin/bash
if ! command -v git-sv &> /dev/null
then
    echo "git-sv could not be found"
    echo "run 'go install github.com/bvieira/sv4git/v2/cmd/git-sv@latest' to install"
    exit
fi

current=v$(git-sv cv)
next=v$(git-sv nv)

printf "\nCurrent:\t$current\n"
printf "Next:\t\t$next\n\n"
{
    git-sv cgl --add-next-version true > CHANGELOG.txt
} || {
    printf "Error creating changelog\n"
    exit
}
git add CHANGELOG.txt
printf "\nTags and CHANGELOG.txt created\n"
printf "Run \n\tgit commit -m \"Update CHANGELOG for $next release\" && git tag $next && git push origin $next\n\nto release\n"
