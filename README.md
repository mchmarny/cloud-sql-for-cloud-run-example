# cloud-sql-for-cloud-run-example

The current Cloud SQL integration in Cloud Run is not yet 100% idiomatic and does require couple GCP-specific steps:

1. Side-effects import `_ "github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/dialers/mysql"`
2. Needing to run [Cloud SQL Proxy](https://cloud.google.com/sql/docs/mysql/quickstart-proxy-test) during local development or testing

This sample outlines a process of setting up a highly available (multi-zone) Cloud SQL instance with secure (TLS) access that works the same way from the developer workstation as it does from within Cloud Run.

> Note, to keep this readme short, I will be asking you to execute scripts rather than listing here complete commands. You should really review each one of these scripts for content, and, to understand the individual commands so you can use them in the future.

## Pre-requirements

### GCP Project and gcloud SDK

If you don't have one already, start by creating new project and configuring [Google Cloud SDK](https://cloud.google.com/sdk/docs/). Similarly, if you have not done so already, you will have [set up Cloud Run](https://cloud.google.com/run/docs/setup).

## API

In case you have not used some of the required GCP APIs, run [bin/api](bin/api) script to make sure they are all enabled:


```shell
bin/api
```

## Setup

To setup this service you will need to clone this repo:

```shell
git clone https://github.com/mchmarny/logo-identifier.git
```

And navigate into that directory:

```shell
cd logo-identifier
```

## Cloud SQL

### Passwords

The [bin/password](bin/password) script will generate root and app user passwords and saved them in a project scoped path under `.cloud-sql` folder in your home directory.

```shell
bin/password
```

### Instance

The [bin/instance](bin/instance) script will:

* Create a Cloud SQL instance
* Set the default (root) user credentials
* Configure MySQL database in the new Cloud SQL instance
* Set up application database user and its credentials
* Create and download client SSL certificates from the newly created instance

> Note, while the created Cloud SQL instance will be exposed to the world (`0.0.0.0/0`), it allow only SSL connections. Also, the root and app user passwords created in first step. If you ever decide to remove the SSL connection requirements, you can reset the root password in the Cloud SQL UI.

```shell
bin/instance
```

### Schema

The [bin/schema](bin/schema) script applies database schema located in [sql/schema.ddl](sql/schema.ddl).

> The provided script checks for existence of all the objects before creating them so you can run it multiple times. it only creates one simple table right now so feel free to edit it before executing the schema script

```shell
bin/schema
```

### Test Connection

At this point you should be able to connect to the newly created database with this command:

```shell
bin/connect
```

### Certificates

The [bin/secret](bin/secret) script creates KMS keys, encrypts Cloud SQL certificates, and save them to a GCS bucket so that the Cloud Run service can securely obtain them while connecting to Cloud SQL DB

```shell
bin/secret
```

## Cloud Run

Once the Cloud SQL instance is configured, you can now deploy the Cloud Run service. First though, you will have to build the image and create a specific service account under which the new service will run.

### Container Image

First, build container image from the included source using the [bin/image](bin/image) script

```shell
bin/image
```

### Service Account

After that, create a service account and assign it the necessary roles using the [bin/user](bin/user) script

```shell
bin/user
```

### Managed Service Deployment

Once the container image and service account are ready, you can deploy the new service using [bin/service](bin/service) script

```shell
bin/service
```

### Cloud Run

At this point you should be able to access your deployed service.

> Note, there is currently no way tp predict the service URL, specifically the bit between the service name (`cloudsql-demo`) and the static Cloud Run domain (`uc.a.run.app`).


Now, navigate in browser to the service URL which will return a JSON response.

```json
{
    "request_id":  "1224d739-cfa5-4500-9a8e-97df6a583aee",
    "request_on":  "2019-08-19 21:14:58.565436028 +0000 UTC",
    "info":        "Success - records saved: 1"
}
```

If for some reason there were errors while inviting the service, the response will include the error details in the `info` field.

### Run Service Locally

You can run the sample service locally by executing the [bin/run](bin/run) script

```shell
bin/run
```

And navigating to http://localhost:8080/v1/test

## Disclaimer

This is my personal project and it does not represent my employer. I take no responsibility for issues caused by this code. I do my best to ensure that everything works, but if something goes wrong, my apologies is all you will get.
