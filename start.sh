#!/bin/sh

# start the spitty executable
./spito 2> /home/lambros/public/spito/log/stderr_spito.log 1> /home/lambros/public/spito/log/stdout_spito.log &

# log its PID for easy temrination
echo $! > RUNNING_PID

