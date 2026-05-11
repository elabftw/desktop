# lint everything in precommit
fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

lint:
	go vet ./...
	cd frontend && npm run lint
