#!/bin/bash

# Check if the correct number of arguments is provided.
if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <log_file> <fb|lb|fe|le>"
    exit 1
fi

log_file="$1"
argument="$2"

# Validate the argument.
case "$argument" in
    "fb"|"lb"|"fe"|"le")
        ;;
    *)
        echo "Invalid argument. Use 'fb' for First Block, 'lb' for Last Block, 'fe' for First Epoch, or 'le' for Last Epoch."
        exit 1
        ;;
esac

# Extract the value based on the argument.
case "$argument" in
    "fb")
        value=$(grep -oP "First Block: \K\d+" "$log_file")
        ;;
    "lb")
        value=$(grep -oP "Last Block: \K\d+" "$log_file")
        ;;
    "fe")
        value=$(grep -oP "First Epoch: \K\d+" "$log_file")
        ;;
    "le")
        value=$(grep -oP "Last Epoch: \K\d+" "$log_file")
        ;;
esac

echo "$value"
