#!/bin/sh

# Retrieve the JSON data from the website
json=$(curl -s https://go.dev/dl/?mode=json)

# Extract the Go versions from the JSON data
versions=$(echo "$json" | jq -r '.[].version')

# Set an environment variable with the Go versions
export GO_VERSIONS="$versions"
