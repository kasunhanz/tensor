#!/bin/bash
set -m

/opt/mongo/mongo_setup_users.sh

mongodb_cmd="/usr/bin/mongod --storageEngine $STORAGE_ENGINE"
cmd="$mongodb_cmd --httpinterface --rest"

if [ "$AUTH" == "yes" ]; then
  cmd="$cmd --auth"
fi

if [ "$JOURNALING" == "no" ]; then
  cmd="$cmd --nojournal"
fi

if [ "$OPLOG_SIZE" != "" ]; then
  cmd="$cmd --oplogSize $OPLOG_SIZE"
fi

if [ "$MONGO_DB_PATH" != "" ]; then
  if [ ! -d "$MONGO_DB_PATH" ]; then
    echo "Creating custom directory for MongoDB data at $MONGO_DB_PATH"
    mkdir -p $MONGO_DB_PATH
  fi
  cmd="$cmd --dbpath $MONGO_DB_PATH"
fi

$cmd &

fg
