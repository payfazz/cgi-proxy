# cgi-proxy

A simple server to run CGI script. This is useful for adding ad-hoc integration for your environment.

## How to run

### Create configuration file

create `config.yml`, for example:

```yaml
static_key:
- 3009b87324da39e76f60a85fe030b2f8
- 9cd7dd0873172eda537bafe618ec72b4

entry:
- path: /test1
  cmd: ["./cgify.sh", "./test-script.sh", "a", "b"]
- path: /test2
  cmd: ["./cgify.sh", "./test-script.sh", "1", "2"]

```

`static_key` is used as username for HTTP Basic credential (with empty password), it is global configuration, credential per script is not supported yet.

### Running the server

then run the server

```sh
APP_LISTEN=:8080 APP_CONFIG=./config.yaml ./cgi-proxy
```

you can omit `APP_LISTEN` and `APP_CONFIG` environment variable, the default value is `:8080` and `./config.yaml`.

### Access the CGI script

```sh
curl http://3009b87324da39e76f60a85fe030b2f8@localhost:8080/test1
curl http://9cd7dd0873172eda537bafe618ec72b4@localhost:8080/test2
```

### Reload configuration

```sh
kill -s HUP "$CGI_PROXY_PID"
```

## Example usecase

Create a script for deploying from github, it can be bash or python. Then use this project for gluing github-webhook to that scripts.
