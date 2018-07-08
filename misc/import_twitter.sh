#!/usr/bin/env bash

set +e
set -x

BASE_DIR="/sites/mdb"
TIMESTAMP="$(date '+%Y%m%d%H%M%S')"
LOG_FILE="$BASE_DIR/logs/twitter/import_$TIMESTAMP.log"

cd ${BASE_DIR} && ./mdb twitter-latest > ${LOG_FILE} 2>&1

WARNINGS="$(egrep -c "level=(warning|error)" ${LOG_FILE})"

if [ "$WARNINGS" = "" ];then
        echo "No warnings"
        exit 0
fi

echo "Errors in periodic import of twitter to MDB" | mail -s "ERROR: MDB twitter import" -r "mdb@bbdomain.org" -a ${LOG_FILE} edoshor@gmail.com

find "${BASE_DIR}/logs/twitter/import_*.log" -type f -mtime +7 -exec rm -rf {} \;
