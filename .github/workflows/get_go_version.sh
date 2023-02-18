#!/bin/sh

# Retrieve the JSON data from the website
json=$(curl -s https://go.dev/dl/?mode=json)

# Extract Go versions from JSON data
versions=$(echo "$json" | jq -r '.[].version' | sed 's/^go//')
echo "export GO_VERSIONS=\"$versions\"" >> ~/.bashrc

# set environment variable
export GO_VERSIONS="$versions"