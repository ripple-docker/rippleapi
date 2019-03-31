#!/bin/sh

sed -i 's/OSUAPIKEY/'"$OSUAPIKEY"'/g' api.conf
sed -i 's/MYSQL_ROOT_PASSWORD/'"$MYSQL_ROOT_PASSWORD"'/g' api.conf
exec "$@"
