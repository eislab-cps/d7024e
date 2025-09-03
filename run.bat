@echo off
cd labs
docker swarm init && docker stack deploy --detach=false -c docker-compose.yml nodestack
cd ..