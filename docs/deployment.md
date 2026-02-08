# Deployment

Step-by-step guide to deploy the collaborative editor on Google Cloud Run with Firestore persistence.

## Prerequisites

- [Google Cloud CLI](https://cloud.google.com/sdk/docs/install) installed
- Authenticated: `gcloud auth login`

## Step 1: Create a GCP project

Create a new project (or skip if using an existing one):

```bash
gcloud projects create go-collab-editor --name="Collab Editor"
gcloud config set project go-collab-editor
```

Enable billing on the project — required for Cloud Run and Firestore, but the free tier covers typical personal use.

## Step 2: Enable APIs

```bash
gcloud services enable \
  firestore.googleapis.com \
  run.googleapis.com \
  cloudbuild.googleapis.com
```

- **Firestore** — document storage
- **Cloud Run** — serverless container hosting
- **Cloud Build** — builds your Dockerfile when deploying with `--source`

## Step 3: Create the Firestore database

```bash
gcloud firestore databases create --location=nam5
```

`nam5` is the multi-region US location (free tier eligible). Choose a location close to your Cloud Run region. Other options:

| Location | Description |
|----------|-------------|
| `nam5` | Multi-region US |
| `eur3` | Multi-region Europe |
| `us-central1` | Single region — Iowa |
| `europe-west1` | Single region — Belgium |

## Step 4: Deploy to Cloud Run

From the project root directory:

```bash
gcloud run deploy go-collab-editor \
  --source . \
  --region us-central1 \
  --allow-unauthenticated \
  --set-env-vars GCP_PROJECT=go-collab-editor \
  --args="-store=firestore"
```

What each flag does:

- `--source .` — builds from the Dockerfile via Cloud Build
- `--region us-central1` — where the service runs (pick the same region as Firestore for lowest latency)
- `--allow-unauthenticated` — makes the editor publicly accessible
- `--set-env-vars GCP_PROJECT=go-collab-editor` — tells the app which Firestore project to use
- `--args="-store=firestore"` — selects the Firestore storage backend

Cloud Run automatically sets the `PORT` environment variable, which the app reads.

## Step 5: Open the URL

The deploy command prints a service URL like:

```
Service URL: https://go-collab-editor-xxxxx-uc.a.run.app
```

Open it in your browser. Share the URL to collaborate in real-time.

## Updating

To deploy a new version after code changes:

```bash
gcloud run deploy go-collab-editor --source . --region us-central1
```

The env vars and args from the previous deployment are preserved.

## Authentication

Cloud Run uses Application Default Credentials. The default Compute Engine service account automatically has Firestore access — no service account key files needed.

If you've restricted IAM permissions, ensure the Cloud Run service account has the **Cloud Datastore User** role:

```bash
gcloud projects add-iam-policy-binding go-collab-editor \
  --member="serviceAccount:$(gcloud iam service-accounts list --format='value(email)' --filter='displayName:Compute Engine')" \
  --role="roles/datastore.user"
```

## Free tier limits

| Resource | Free tier |
|----------|-----------|
| Cloud Run | 2M requests/month, 360K vCPU-seconds, 180K GiB-seconds |
| Firestore | 1 GB storage, 50K reads/day, 20K writes/day |
| Cloud Build | 120 build-minutes/day |
