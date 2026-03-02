#!/bin/sh
printf "Enter password: "
stty -echo
read password
stty echo
printf "\n"
echo -n "$password" | sha256sum | awk '{print $1}'
