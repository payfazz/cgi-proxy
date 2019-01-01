#!/bin/sh

echo "args:"
for arg; do
  echo $arg
done
echo ""

echo "env:"
env | sort
echo ""

echo "body:"
cat
echo ""
