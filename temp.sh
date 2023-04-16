#!/bin/sh

terrad q staking validator $(terrad keys show test0 -a --bech val --keyring-backend test --home $HOME) >/dev/null 2>&1

if [ $? -eq 0 ]; then
    echo "Success"
else
    echo "Failed"
fi