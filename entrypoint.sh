#!/bin/sh

sed -i 's/OSUAPIKEY/'"$OSUAPIKEY"'/g' api.conf
exec "$@"
