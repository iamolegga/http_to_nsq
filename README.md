<h1 align="center">http_to_nsq</h1>
<p align="center">Forward `POST /:topic` to NSQ</p>

<p align="center">
  <a href="https://hub.docker.com/r/iamolegga/http_to_nsq">
    <img alt="Docker Image Version (latest semver)" src="https://img.shields.io/docker/v/iamolegga/http_to_nsq?sort=semver">
  </a>
  <a href="https://github.com/iamolegga/http_to_nsq/actions/workflows/on-push-main.yml?query=branch%3Amain">
    <img alt="GitHub Workflow Status (with branch)" src="https://img.shields.io/github/actions/workflow/status/iamolegga/http_to_nsq/on-push-main.yml?branch=main">
  </a>
  <a href="https://snyk.io/test/github/iamolegga/http_to_nsq">
    <img alt="Snyk Vulnerabilities for GitHub Repo (Specific Manifest)" src="https://img.shields.io/snyk/vulnerabilities/github/iamolegga/http_to_nsq/go.mod" />
  </a>
  <a href="https://libraries.io/github/iamolegga/http_to_nsq">
    <img alt="Libraries.io dependency status for GitHub repo" src="https://img.shields.io/librariesio/github/iamolegga/http_to_nsq" />
  </a>
  <img alt="Dependabot" src="https://badgen.net/github/dependabot/iamolegga/http_to_nsq" />
  <img alt="Docker Pulls" src="https://img.shields.io/docker/pulls/iamolegga/http_to_nsq" />
</p>

## Usage

```
$ ./http_to_nsq --help
Usage of ./http_to_nsq:
  -gom
    	Expose Go runtime metrics
  -log value
    	log level (debug, info, warn, error, dpanic, panic, fatal)
  -lookupd-http-address string
    	nsqlookupd HTTP address
  -nsqd-tcp-address string
    	nsqd TCP address (default "localhost:4150")
  -port int
    	HTTP port (default 4252)
```

Or in docker:

```shell
docker run --rm -p 4252:4252 iamolegga/http_to_nsq \
  -nsqd-tcp-address=host.docker.internal:4150
```

Metrics are exposed at `/metrics` endpoint on the same port as the HTTP server.
