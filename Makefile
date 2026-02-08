.PHONY: docs-serve docs-build docs-api docs-deploy docs-setup test run \
       gcp-setup gcp-deploy gcp-destroy

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

# GCP project and region (override: make gcp-deploy GCP_PROJECT=my-project GCP_REGION=europe-west1)
GCP_PROJECT ?= $(shell gcloud config get-value project 2>/dev/null)
GCP_REGION  ?= us-central1

# One-time GCP setup: enable APIs and create Firestore database
gcp-setup:
	gcloud services enable firestore.googleapis.com run.googleapis.com cloudbuild.googleapis.com
	gcloud firestore databases create --location=nam5 || true

# Deploy to Cloud Run with Firestore backend
gcp-deploy:
	gcloud run deploy go-collab-editor \
		--source . \
		--region $(GCP_REGION) \
		--allow-unauthenticated \
		--set-env-vars GCP_PROJECT=$(GCP_PROJECT) \
		--args="-store=firestore"

# Tear down Cloud Run service and Firestore database
gcp-destroy:
	gcloud run services delete go-collab-editor --region $(GCP_REGION) --quiet
	gcloud firestore databases delete --database='(default)' --quiet
