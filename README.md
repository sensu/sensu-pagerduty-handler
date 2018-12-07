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
        "command": "sensu-pagerduty-handler --token SECRET",
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
  -h, --help           help for sensu-pagerduty-handler
  -t, --token string   The PagerDuty V2 API authentication token
```

## Contributing

See https://github.com/sensu/sensu-go/blob/master/CONTRIBUTING.md

[1]: https://github.com/sensu/sensu-go
[2]: https://www.pagerduty.com/ 
[3]: https://docs.sensu.io/sensu-go/5.0/reference/handlers/#how-do-sensu-handlers-work
[4]: https://github.com/sensu/sensu-pagerduty-handler/releases
