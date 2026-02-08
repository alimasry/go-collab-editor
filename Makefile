.PHONY: docs-serve docs-build docs-api docs-deploy docs-setup test run

# Install documentation tooling
docs-setup:
	pip install -r requirements.txt
	go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest

# Regenerate API reference from Go doc comments
docs-api:
	gomarkdoc --output docs/api/ot.md ./ot/
	gomarkdoc --output docs/api/server.md ./server/
	gomarkdoc --output docs/api/store.md ./store/

# Serve docs locally with hot-reload (http://localhost:8000)
docs-serve: docs-api
	mkdocs serve

# Build static site to site/ directory
docs-build: docs-api
	mkdocs build --strict

# Deploy to GitHub Pages (pushes to gh-pages branch)
docs-deploy: docs-api
	mkdocs gh-deploy --force

# Run all Go tests
test:
	go test ./...

# Run the server
run:
	go run main.go
