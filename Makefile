.PHONY: run

run:
	@set -a; \
	if [ -f .env ]; then . ./.env; fi; \
	set +a; \
	go run ./cmd/api
