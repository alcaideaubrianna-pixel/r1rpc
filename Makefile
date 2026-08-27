COMPOSE := docker compose -f deploy/docker-compose.yml

.PHONY: dev-deps dev dev-full dev-stop dev-clean dev-status dev-logs

dev-deps:
	$(COMPOSE) up -d --pull never mysql redis

dev: dev-deps
	JWT_SECRET=$${JWT_SECRET:-r1rpc-local-dev-secret-change-me} \
	BOOTSTRAP_ADMIN_PASSWORD=$${BOOTSTRAP_ADMIN_PASSWORD:-123456} \
	HTTP_ADDR=$${HTTP_ADDR:-:9876} \
	MYSQL_HOST=$${MYSQL_HOST:-127.0.0.1} \
	MYSQL_PORT=$${R1RPC_MYSQL_PORT:-3306} \
	MYSQL_USER=$${MYSQL_USER:-root} \
	MYSQL_PASSWORD=$${MYSQL_PASSWORD:-r1rpc_dev} \
	MYSQL_DB=$${MYSQL_DATABASE:-r1rpc} \
	air -c .air.toml

dev-full:
	$(COMPOSE) --profile full up -d --build --pull never

dev-stop:
	$(COMPOSE) --profile full down

dev-clean:
	$(COMPOSE) --profile full down -v

dev-status:
	$(COMPOSE) --profile full ps

dev-logs:
	$(COMPOSE) --profile full logs -f
