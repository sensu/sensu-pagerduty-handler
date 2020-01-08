# Sensu Go PagerDuty Handler

[![Bonsai Asset Badge](https://img.shields.io/badge/Bonsai-Download%20Me-brightgreen.svg?colorB=89C967&logo=sensu)](https://bonsai.sensu.io/assets/sensu/sensu-pagerduty-handler)
[![ Build Status](https://travis-ci.org/sensu/sensu-pagerduty-handler.svg?branch=master)](https://travis-ci.org/sensu/sensu-pagerduty-handler)

- [Overview](#overview)
- [Usage examples](#usage-examples)
- [Configuration](#configuration)
  - [Environment Variables](#environment-variables)
- [Setup](#setup)
- [Examples](#examples)

## Overview

The Sensu Go PagerDuty Handler is a [Sensu Go event handler][3] which manages
[PagerDuty][2] incidents for alerting operators. With this handler,
[Sensu][1] can trigger and resolve PagerDuty incidents.

## Usage examples

Help:
```
Usage:
  sensu-pagerduty-handler [flags]
Flags:
  -d, --dedup-key string            The Sensu event label specifying the PagerDuty V2 API deduplication key, use default from PAGERDUTY_DEDUP_KEY env var
  -k, --dedup-key-template string   The PagerDuty V2 API deduplication key template, use default from PAGERDUTY_DEDUP_KEY_TEMPLATE env var
  -h, --help                        help for sensu-pagerduty-handler
  -s, --status-map string           The status map used to translate a Sensu check status to a PagerDuty severity, use default from PAGERDUTY_STATUS_MAP env var
  -t, --token string                The PagerDuty V2 API authentication token, use default from PAGERDUTY_TOKEN env var
```

## Configuration

You can use either command line flags or environment variables to configure the Sensu Pagerduty Handler. The environment variables can be in either the handler definition or in the Sensu backend service environment.

**Note:** Make sure to set the `PAGERDUTY_TOKEN` environment variable for sensitive credentials in production to prevent leaking into system process table. Please remember command arguments can be viewed by unprivileged users using commands such as `ps` or `top`. The `--token` argument is provided as an override primarily for testing purposes.

Pagerduty Token        | |
---------------------|-------------------------------
description          | The PagerDuty V2 API authentication token.
long                 | token
short                | t
env variable         | PAGERDUTY_TOKEN
required             | true
type                 | String
default              | null
example              | `PAGERDUTY_TOKEN=f88b6c03e43d3aa939959a8f47fjge`

Pagerduty Dedup Key        | |
---------------------|-------------------------------
description          | The Sensu event label specifying the PagerDuty V2 API deduplication key.
long                 | dedup-key
short                | d
env variable         | PAGERDUTY_DEDUP_KEY
required             | false
type                 | String
default              | null
example              | `PAGERDUTY_DEDUP_KEY=test_label`

Pagerduty Dedup Key Template        | |
---------------------|-------------------------------
description          | The PagerDuty V2 API deduplication key template.
long                 | dedup-key-template
short                | k
env variable         | PAGERDUTY_DEDUP_KEY_TEMPLATE
required             | false
type                 | String
default              | null
example              | `PAGERDUTY_DEDUP_KEY_TEMPLATE={{.Entity.Name}}-{{.Check.Name}}`

Pagerduty Token        | |
---------------------|-------------------------------
description          | The status map used to translate a Sensu check status to a PagerDuty severity.
long                 | status-map
short                | s
env variable         | PAGERDUTY_STATUS_MAP
required             | false
type                 | Map
default              | null
example              | `PAGERDUTY_STATUS_MAP={\"info\":[130,10],\"error\":[4]}`

### Deduplication Key Priority

The deduplication key is determined using the following priority:
1. --dedup-key  --  specifies the entity label containing the key
1. --dedup-key-template  --  a template containing the values
1. the default value containing the entity and check names

### PagerDuty Severity Mapping

Optionally you can provide mapping information between the Sensu check status and the PagerDuty incident severity.
To provide the mapping you need to use the `--status-map` command line option or the `PAGERDUTY_STATUS_MAP` environment variable.
The option accepts a JSON document containing the mapping information. Here's an example of the JSON document:

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

The valid [PagerDuty alert severity levels][5] are the following:
* `info`
* `warning`
* `critical`
* `error`

## Setup

1. Register the pagerduty handler

   ```shell
   sensuctl asset add sensu/sensu-pagerduty-handler
   ```

2. Configure the pagerduty handler.

   ```yaml
   ---
   type: Handler
   api_version: core/v2
   metadata:
     name: pagerduty
     namespace: default
   spec:
     type: pipe
     command: sensu-remediation-handler
     timeout: 10
     runtime_assets:
     - sensu-remediation-handler
     env_vars:
     - "SENSU_API_URL=http://127.0.0.1:8080"
     - "SENSU_API_CERT_FILE="
     - "SENSU_API_USER=remediation-handler"
     - "SENSU_API_PASS=supersecret"
   ```

   Save this definition to a file named `sensu-pagerduty-handler.yaml` and
   run:

   ```shell
   sensuctl create -f sensu-pagerduty-handler.yaml
   ```

## Examples

### Example Sensu Go Check Definition to use the Sensu Go Pagerduty Handler:

```yaml
---
api_version: core/v2
type: CheckConfig
metadata:
  namespace: default
  name: dummy-app-healthz
spec:
  command: check-http -u http://localhost:8080/healthz
  subscriptions:
  - dummy
  publish: true
  interval: 10
  handlers:
  - pagerduty

```

### Deduplication Key Priority

The deduplication key is determined using the following priority:
1. --dedup-key  --  specifies the entity label containing the key
1. --dedup-key-template  --  a template containing the values
1. the default value containing the entity and check names

### PagerDuty Severity Mapping

Optionally you can provide mapping information between the Sensu check status and the PagerDuty incident severity.
To provide the mapping you need to use the `--status-map` command line option or the `PAGERDUTY_STATUS_MAP` environment variable.
The option accepts a JSON document containing the mapping information. Here's an example of the JSON document:

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

The valid [PagerDuty alert severity levels][5] are the following:
* `info`
* `warning`
* `critical`
* `error`


[1]: https://github.com/sensu/sensu-go
[2]: https://www.pagerduty.com/ 
[3]: https://docs.sensu.io/sensu-go/5.0/reference/handlers/#how-do-sensu-handlers-work
[4]: https://github.com/sensu/sensu-pagerduty-handler/releases
[5]: https://support.pagerduty.com/docs/dynamic-notifications#section-eventalert-severity-levels
