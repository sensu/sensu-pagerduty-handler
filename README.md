<div class=badges>
  <a href="https://bonsai.sensu.io/assets/sensu/sensu-pagerduty-handler">
    <img src="https://img.shields.io/badge/Sensu%20PagerDuty%20Handler-Download%20Me-brightgreen.svg?colorB=89C967&logo=sensu" alt="Bonsai Asset Badge">
  </a>
  <a href="https://github.com/sensu/sensu-pagerduty-handler/actions?query=workflow%3A%22Go+Test%22">
    <img src="https://github.com/sensu/sensu-pagerduty-handler/workflows/Go%20Test/badge.svg" alt="Go Test Actions Workflow">
  </a>
  <a href="https://github.com/sensu/sensu-pagerduty-handler/actions?query=workflow%3Agoreleaser">
    <img src="https://github.com/sensu/sensu-pagerduty-handler/workflows/goreleaser/badge.svg" alt="Goreleaser Actions Workflow">
  </a>
</div>

# Sensu PagerDuty Handler

## Table of Contents
- [Overview](#overview)
- [Quick start](#quick-start)
- [Usage](#usage)
  - [Help](#help)
  - [Environment variables](#environment-variables)
  - [Argument annotations](#argument-annotations)
  - [Proxy support](#proxy-support)
  - [Deduplication key](#deduplication-key)
  - [PagerDuty severity mapping](#pagerduty-severity-mapping)
- [Configuration](#configuration)
  - [Asset registration](#asset-registration)
  - [Asset definition](#asset-definition)
  - [Handler definition](#handler-definition)
- [Install from source](#install-from-source)
- [Contribute](#contribute)

## Overview

The Sensu PagerDuty Handler is a [Sensu event handler][3] that manages [PagerDuty][2] incidents for alerting operators.
With this handler, [Sensu][1] can trigger and resolve PagerDuty incidents.

## Quick start

We recommend installing the Sensu PagerDuty Handler plugin via the [monitoring-pipelines PagerDuty template](https://github.com/sensu-community/monitoring-pipelines/blob/master/incident-management/pagerduty.yaml).
The template includes Sensu resource definitions for the handler, the versioned asset you will need, and information about supported options.
Make sure to edit the template to match your configuration before you install it with `sensuctl create`.

**NOTE**: The monitoring-pipelines and monitoring-checks templates use specially defined handler sets by default.
For more information about how these templates work, read the [monitoring pipelines readme](https://github.com/sensu-community/monitoring-pipelines/blob/master/README.md).

## Usage

### Help

```
The Sensu Go PagerDuty handler for incident management

Usage:
  sensu-pagerduty-handler [flags]
  sensu-pagerduty-handler [command]

Available Commands:
  help        Help about any command
  version     Print the version number of this plugin

Flags:
  -t, --token string                The PagerDuty V2 API authentication token, can be set with PAGERDUTY_TOKEN
  -k, --dedup-key-template string   The PagerDuty V2 API deduplication key template, can be set with PAGERDUTY_DEDUP_KEY_TEMPLATE (default "{{.Entity.Name}}-{{.Check.Name}}")
  -S, --summary-template string     The template for the alert summary, can be set with PAGERDUTY_SUMMARY_TEMPLATE (default "{{.Entity.Name}}/{{.Check.Name}} : {{.Check.Output}}")
  -s, --status-map string           The status map used to translate a Sensu check status to a PagerDuty severity, can be set with PAGERDUTY_STATUS_MAP
  -h, --help                        help for sensu-pagerduty-handler

Use "sensu-pagerduty-handler [command] --help" for more information about a command.

```

### Environment variables

You can set most arguments for the Sensu PagerDuty Handler via environment variables.
However, any arguments specified directly on the command line will override the corresponding environment variable.

|Argument            |Environment Variable          |
|--------------------|------------------------------|
|--token             |`PAGERDUTY_TOKEN`             |
|--dedup-key-template|`PAGERDUTY_DEDUP_KEY_TEMPLATE`|
|--summary-template  |`PAGERDUTY_SUMMARY_TEMPLATE`  |
|--status-map        |`PAGERDUTY_STATUS_MAP`        |

**IMPORTANT**: Take care to avoid exposing the handler authentication token by specifying it on the command line or directly setting the `PAGERDUTY_TOKEN`
environment variable in the handler definition.
Consider using [secrets management][8] to surface the token as an environment variable as shown in the [handler definition][10].

To use Sensu's built-in [env secrets provider][9] to set the `PAGERDUTY_TOKEN` environment variable:

```yml
---
type: Secret
api_version: secrets/v1
metadata:
  name: pagerduty_token
spec:
  provider: env
  id: PAGERDUTY_TOKEN
```

### Argument annotations

You can tune all arguments for the Sensu PagerDuty Handler on a per entity or per check basis based on annotations.
The annotations keyspace for the handler is `sensu.io/plugins/sensu-pagerduty-handler/config`.

#### Annotation example

To change the token argument for a particular check, add this annotation to the checks's metadata:

```yml
type: CheckConfig
api_version: core/v2
metadata:
  annotations:
    sensu.io/plugins/sensu-pagerduty-handler/config/token: abcde12345fabcd67890efabc12345de
[...]
```

### Proxy support

The Sensu PagerDuty Handler supports the environment variables `HTTP_PROXY`, `HTTPS_PROXY`, and `NO_PROXY` (or the lowercase versions thereof).
`HTTPS_PROXY` takes precedence over `HTTP_PROXY` for HTTPS requests.

The environment values may be a complete URL or a "host[:port]".
If you use a "host[:port]", the "http" scheme is assumed.

### Deduplication key

The deduplication key is determined via the `--dedup-key-template` argument.
The key is a Golang template that contains the event values and defaults to `{{.Entity.Name}}-{{.Check.Name}}`.

### PagerDuty severity mapping

If you wish, you can provide mapping information between the Sensu check status and the PagerDuty incident severity.
To do this, use the `--status-map` command line option or the `PAGERDUTY_STATUS_MAP` environment variable.

The valid [PagerDuty alert severity levels][5] are:
* `info`
* `warning`
* `critical`
* `error`

Use a JSON document to provide the mapping information.
For example:

```json
{
    "info": [
        0,
        1
    ],
    "warning": [
        2
    ],
    "critical:": [
        3
    ],
    "error": [
        4,
        5,
        6,
        7,
        8,
        9,
        10
    ]
}
```

## Configuration

**NOTE**: If you use the [monitoring-pipelines PagerDuty template](https://github.com/sensu-community/monitoring-pipelines/blob/master/incident-management/pagerduty.yaml), the template provides the asset and handler resource definitions you need.
Read the inline comments in the template for information about template configuration options.

### Asset registration

[Sensu assets][6] are the best way to install and use this plugin.
If you're not using an asset, please consider doing so!

If you're using sensuctl 5.13 with Sensu backend 5.13 or later, you can use the following command to add the asset:

```
sensuctl asset add sensu/sensu-pagerduty-handler
```

If you're using an earlier version of sensuctl, you can find the asset on [Bonsai, the Sensu asset index][7].

### Asset definition

This is an example of a manually created asset definition:

```yml
---
type: Asset
api_version: core/v2
metadata:
  name: sensu-pagerduty-handler_linux_amd64
spec:
  url: https://assets.bonsai.sensu.io/e930fc9c21b835896216ca4594c7990111b54630/sensu-pagerduty-handler_2.0.1_linux_amd64.tar.gz 
  sha512: 6d499ae6edeb910eb807abfb141be3b9a72951911f804d5a7cc98fbbea15dc4cc6c456f1663124c1db51111c8fd42dab39d1d093356e555785f21ef5f95ffb06
  filters:
  - entity.system.os == 'linux'
  - entity.system.arch == 'amd64'
```

You can use this definition with `sensu create` to register a version of this asset without using the Bonsai integration.

### Handler definition

This is an example of a manually created handler definition:

```yml
---
type: Handler
api_version: core/v2
metadata:
  name: pagerduty
  namespace: default
spec:
  type: pipe
  command: sensu-pagerduty-handler
  timeout: 10
  runtime_assets:
  - sensu/sensu-pagerduty-handler
  filters:
  - is_incident
  secrets:
  - name: PAGERDUTY_TOKEN
    secret: pagerduty_authtoken
```

## Install from source

Download the latest version of the sensu-pagerduty-handler from [releases][4] or create an executable script from this source.

From the local path of the sensu-pagerduty-handler repository, run:

```
go build -o /usr/local/bin/sensu-pagerduty-handler
```

## Contribute

See https://github.com/sensu/sensu-go/blob/master/CONTRIBUTING.md to contribute to this project.


[1]: https://github.com/sensu/sensu-go
[2]: https://www.pagerduty.com/
[3]: https://docs.sensu.io/sensu-go/5.0/reference/handlers/
[4]: https://github.com/sensu/sensu-pagerduty-handler/releases
[5]: https://support.pagerduty.com/docs/dynamic-notifications#section-eventalert-severity-levels
[6]: https://docs.sensu.io/sensu-go/latest/operations/deploy-sensu/assets/
[7]: https://bonsai.sensu.io/assets/sensu/sensu-pagerduty-handler
[8]: https://docs.sensu.io/sensu-go/latest/operations/manage-secrets/secrets-management/
[9]: https://docs.sensu.io/sensu-go/latest/operations/manage-secrets/secrets-management/#use-env-for-secrets-management
[10]: #handler-definition
