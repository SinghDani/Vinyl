#!/bin/sh
#Todo change this here
##!/bin/bash
shopt -s nullglob
#go build -o fingerprinting
for file in audioFiles/*.wav; do
    echo "$file"
    #./fingerprinting 1 "$file"
    ./audioRec "store" "$file"
done

#rm fingerprinting
