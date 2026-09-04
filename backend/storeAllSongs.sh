#!/bin/bash

#Todo change this here
shopt -s nullglob
go build -o audioRec
for file in audioFiles/*.wav; do
    echo "$file"
    ./audioRec "store" "$file"
done

rm audioRec
