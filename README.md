# Sensu Go PagerDuty Handler
TravisCI: [![TravisCI Build Status](https://travis-ci.org/sensu/sensu-pagerduty-handler.svg?branch=master)](https://travis-ci.org/sensu/sensu-pagerduty-handler)

The Sensu Go PagerDuty Handler is a [Sensu Event Handler][3] which manages
[PagerDuty][2] incidents, for alerting operators. With this handler,
[Sensu][1] can trigger and resolve PagerDuty incidents.

## Installation

Download the latest version of the sensu-pagerduty-handler from [releases][4],
or create an executable script from this source.

From the local path of the sensu-pagerduty-handler repository:
```
go build -o /usr/local/bin/sensu-pagerduty-handler main.go
```

## Configuration

Example Sensu Go handler definition:

```json
{
    "api_version": "core/v2",
    "type": "Handler",
    "metadata": {
        "namespace": "default",
        "name": "pagerduty"
    },
    "spec": {
        "type": "pipe",
        "command": "sensu-pagerduty-handler",
        "env_vars": [
          "PAGERDUTY_TOKEN=SECRET",
          "PAGERDUTY_DEDUP_KEY",
          "PAGERDUTY_STATUS_MAP"
        ],
        "timeout": 10,
        "filters": [
            "is_incident"
        ]
    }
}
```

Example Sensu Go check definition:

```json
{
    "api_version": "core/v2",
    "type": "CheckConfig",
    "metadata": {
        "namespace": "default",
        "name": "dummy-app-healthz"
    },
    "spec": {
        "command": "check-http -u http://localhost:8080/healthz",
        "subscriptions":[
            "dummy"
        ],
        "publish": true,
        "interval": 10,
        "handlers": [
            "pagerduty"
        ]
    }
}
```

## Usage Examples

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

**Note:** Make sure to set the `PAGERDUTY_TOKEN` environment variable for sensitive credentials in production to prevent leaking into system process table. Please remember command arguments can be viewed by unprivileged users using commands such as `ps` or `top`. The `--token` argument is provided as an override primarily for testing purposes. 

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

The valid PagerDuty severities are the following:
* `info`
* `warning`
* `critical`
* `error`

## Contributing

See https://github.com/sensu/sensu-go/blob/master/CONTRIBUTING.md

[1]: https://github.com/sensu/sensu-go
[2]: https://www.pagerduty.com/ 
[3]: https://docs.sensu.io/sensu-go/5.0/reference/handlers/#how-do-sensu-handlers-work
[4]: https://github.com/sensu/sensu-pagerduty-handler/releases
