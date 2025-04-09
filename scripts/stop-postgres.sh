docker stop $(docker ps -q --filter ancestor=postgres)
exit 0
