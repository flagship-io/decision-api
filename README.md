<p align="center">
<img  src="https://mk0abtastybwtpirqi5t.kinstacdn.com/wp-content/uploads/picture-solutions-persona-product-flagship.jpg"  width="211"  height="182"  alt="flagship-java"  />
</p>
<h3 align="center">Bring your features to life</h3>

[![codecov](https://codecov.io/gh/flagship-io/decision-api/branch/main/graph/badge.svg?token=Jvuh2U89uA)](https://codecov.io/gh/flagship-io/decision-api)
[![Go Report Card](https://goreportcard.com/badge/github.com/flagship-io/decision-api)](https://goreportcard.com/report/github.com/flagship-io/decision-api)
[![CI](https://github.com/flagship-io/decision-api/actions/workflows/ci.yml/badge.svg)](https://github.com/flagship-io/decision-api/actions/workflows/ci.yml) 

**Visit [https://docs.developers.flagship.io/](https://docs.developers.flagship.io/) to get started with Flagship.**

## Disclaimer
THIS PROJECT IS IN EARLY ADOPTER PHASE. USE AT YOUR OWN RISK.

CONTACT THE FLAGSHIP TEAM FOR MORE INFORMATION

## Docs

### Installation
The Flagship Decision API can be installed and deployed in your infrastructure either by downloading and running the binary, or pulling and running the docker image in your orchestration system.

#### Using a binary
You can download the latest binary here: https://github.com/flagship-io/decision-api/releases

#### Using a Docker image
You can pull the latest docker image from docker hub:
docker pull flagshipio/decision-api

### Running
Using a binary
Download the latest release on github and then simply run:

ENV_ID={your_environment_id} API_KEY={your_api_key} ./decision-api

The server will run on the port 8080 by default. You can override this configuration (see Configuration)

Running with Docker
Run the following command to start the server with Docker

docker run -p 8080:8080 -e ENV_ID={your_env_id} -e API_KEY={your_api_key} flagshipio/decision-api

### Configuration
Full configuration and API options available here:

[https://docs.developers.flagship.io/docs/run-on-premise](https://docs.developers.flagship.io/docs/run-on-premise)

## Licence

[Apache License.](https://github.com/flagship-io/decision-api/blob/main/LICENSE)
