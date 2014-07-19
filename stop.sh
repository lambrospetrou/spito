#!/bin/sh

if [ -e "RUNNING_PID" ] 
then
    cat RUNNING_PID | xargs kill -9 
fi

