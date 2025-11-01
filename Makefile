.PHONY: docker-up docker-down test-e2e

docker-up:
	docker compose up -d --build
	@echo "the server "

docker-down:
	docker compose down -v

test-e2e:
	chmod +x ./e2e-test.sh
	bash ./e2e-test.sh
