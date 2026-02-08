# Deployment

Deploy the collaborative editor on Google Cloud Run with Firestore persistence.

## Prerequisites

- [Google Cloud CLI](https://cloud.google.com/sdk/docs/install) installed
- Authenticated: `gcloud auth login`
- A GCP project with billing enabled

Set your project:

```bash
gcloud config set project YOUR_PROJECT_ID
```

## First-time setup

Enable the required APIs and create the Firestore database:

```bash
make gcp-setup
```

This enables Firestore, Cloud Run, and Cloud Build APIs, and creates a Firestore database in multi-region US (`nam5`).

## Deploy

```bash
make gcp-deploy
```

This builds the Docker image via Cloud Build and deploys to Cloud Run. The service URL is printed when the deploy completes.

To use a different project or region:

```bash
make gcp-deploy GCP_PROJECT=my-project GCP_REGION=europe-west1
```

## Updating

After code changes, redeploy with the same command:

```bash
make gcp-deploy
```

The env vars and args from the previous deployment are preserved.

## Authentication

Cloud Run uses Application Default Credentials. The default Compute Engine service account automatically has Firestore access — no service account key files needed.

If you've restricted IAM permissions, ensure the Cloud Run service account has the **Cloud Datastore User** role:

```bash
gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
  --member="serviceAccount:$(gcloud iam service-accounts list --format='value(email)' --filter='displayName:Compute Engine')" \
  --role="roles/datastore.user"
```

## Free tier limits

| Resource | Free tier |
|----------|-----------|
| Cloud Run | 2M requests/month, 360K vCPU-seconds, 180K GiB-seconds |
| Firestore | 1 GB storage, 50K reads/day, 20K writes/day |
| Cloud Build | 120 build-minutes/day |
