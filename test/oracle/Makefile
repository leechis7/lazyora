.PHONY: env up down status logs shell test run grant-admin

env:
	sh env.sh
up: env
	docker compose up -d --wait --wait-timeout 600 oracle
down:
	docker compose down
status:
	docker compose ps
logs:
	docker compose logs -f oracle
shell:
	docker compose exec oracle sqlplus -L lazyora@//localhost:1521/FREEPDB1
grant-admin:
	docker compose exec -T oracle sqlplus -L -s / as sysdba < grants.sql
test:
	sh test.sh
run: up
	$(MAKE) -C ../.. build-native
	sh run.sh
