docker rm -f unsecure-postgres 2>/dev/null || true
docker ps -q --filter "publish=5432" | xargs -r docker rm -f 2>/dev/null || true
docker run -d --name unsecure-postgres -p 5432:5432 -v /data/:/data/ saichler/unsecure-postgres:latest admin admin admin 5432
