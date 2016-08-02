#!/bin/sh
set -xeo

exec /go/bin/carlos-the-curious --token=$SLACKTOKEN\
                                --database_url=$DATABASE_URL
