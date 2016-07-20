#!/bin/sh
set -xeo

exec /opt/bin/carlos-the-curious --token=$SLACKTOKEN\
                                 --database_url=$DATABASE_URL
