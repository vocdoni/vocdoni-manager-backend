#!/bin/sh

RETVAL=0

function grab() {
    if [ "$1" != "0" ]; then
        RETVAL=$1
    fi
}

echo "### Starting test suite ###"
docker-compose up -d

echo "### Waiting for API to be ready"
sleep 5

echo "### Running tests ###"
docker-compose run test timeout 300 ./managertest --host ws://dvotemanager:8000/api/registry --method=generateTokens --usersNumber=500 --logLevel=info
grab "$?"

docker-compose run test timeout 300 ./managertest --host ws://dvotemanager:8000/api/registry --method=registrationFlow --usersNumber=500 --logLevel=info
grab "$?"

echo "### Post run logs ###"
docker-compose logs --tail 300

echo "### Cleaning environment ###"
docker-compose down -v --remove-orphans

exit $RETVAL
