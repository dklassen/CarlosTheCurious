#!/bin/sh
set -xeo

exec CarlosTheCurious  --token=$SLACKTOKEN\
                       --database_url=$DATABASE_URL
