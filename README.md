# test-api

A super simple, very basic, hacked together test api (for testing).

## Endpoints

[GET /metrics]()
[GET /v1/test/args](#get-command-line-arguments)  
[GET /v1/test/env](#get-environment-variables)  
[GET /v1/test/metadata/task]()  
[GET /v1/test/metadata/stats]()  
[GET /v1/test/metadata/task/stats]()  
[GET /v1/test/mirror]()  
[GET /v1/test/panic]()  
[GET /v1/test/ping]()  
[GET /v1/test/routes]()  
[GET /v1/test/status{?code=XXX}]()  
[GET /v1/test/version]()  

### Get Command Line Arguments

Returns the command line arguments as JSON.

For example, starting with `./test-api foo bar baz --help` will return:

```json
{
  "Args": [
    "foo",
    "bar",
    "baz",
    "--help"
  ]
}
```

### Get Environment Variables

Returns the list of environment variables as JSON.

```json
[
  "PWD=/app",
  "SHELL=/bin/bash",
  "USER=nobody",
  "AWS_DEFAULT_REGION=us-east-1"
]
```

## Author

E Camden Fisher <camden.fisher@yale.edu>

## License

GNU Affero General Public License v3.0 (GNU AGPLv3)
Copyright (c) 2020 Yale University
