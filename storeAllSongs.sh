#!/bin/bash

shopt -s nullglob
go build -o fingerprinting
for file in audioFiles/*.wav; do
    echo "$file"
    ./fingerprinting "$file" 1
done

rm fingerprinting
