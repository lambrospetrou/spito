#!/bin/sh

# start the spitty executable
./spitty 2> /home/lambros/public/lp.gs/log/stderr_lp.gs.log 1> /home/lambros/public/lp.gs/log/stdout_lp.gs.log &

# log its PID for easy temrination
echo $! > RUNNING_PID

