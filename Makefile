
startup-pg-docker:
	docker run \
		--name pg-test \
		-v "$$PWD"/examples/pg/script.sql:/docker-entrypoint-initdb.d/init.sql:ro \
		-e POSTGRES_PASSWORD=pass \
		-e POSTGRES_USER=user \
		-e POSTGRES_DB=testdb \
		-p 5432:5432 \
		-d \
		postgres:15
