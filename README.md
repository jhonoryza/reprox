# REPROX

REPROX is a reverse proxy for http and tcp, insipired from
[JPRQ](https://github.com/azimjohn/jprq)

you can expose your home / private network to the public, share your local
website to the world

you can self hosted the server app using your own domain

## Requirement

- domain or get a [free domain here](./FREE_DOMAIN.md)
- vps / server with public ip
- domain TLS certificates or you can generate free
  [letsencrypt certificate](./LETS_ENCRYPT.md)
- disable server firewall
- make server can accept tcp traffic from all port

## How to use

### Preparation

- if you are using linux you can add to `/etc/environment` file
- or just in terminal run export ...

this is a required environment variables

example :

```
export DOMAIN=labstack.myaddr.io
export DOMAIN_EVENT=labstack.myaddr.io:4321
```

this is an optional environment variable

example :

```
export HTTP_PORT=80
export HTTPS_PORT=443

export TLS_PATH_CERT="/home/ubuntu/letsencrypt/certs/live/labstack.myaddr.io/fullchain.pem"
export TLS_PATH_KEY="/home/ubuntu/letsencrypt/certs/live/labstack.myaddr.io/privkey.pem"
```

if you dont specified port the default value will be used

if you dont specified tls file the https will be disabled automatically

### Compile Server App

```bash
./build.server.sh
```

check bin folder and choose the correct binary according to OS

file is prefix with `server-`

### Compile Client App

```bash
./build.client.sh
```

check bin folder and choose the correct binary according to OS

file is prefix with `client-`

### Notes

- every time the environment is changed, you need to recompile both apps

### Server App Usage

before using server app make sure the required environment variables is exported

replace arch with `mac, linux or windows`

```bash
./bin/server-{arch}
```

### Client App Usage

before using client app make sure the required environment variables is exported

replace arch with `mac, linux or windows`

| option | description                  |
| ------ | ---------------------------- |
| -p     | local port                   |
| -s     | subdomain                    |
| -cname | different domain using CNAME |
| --dir  | directory path               |

For exposing any HTTP servers

```bash
./bin/client-{arch} http -p 3000 -s subdomain
```

For exposing any TCP servers, such as SSH

```bash
./bin/client-{arch} tcp -p 22 -s subdomain
```

Exposing on a different domain

```bash
./bin/client-{arch} http -p 3000 -cname example.com
```

Exposing directory static files using built-in HTTP Server

```bash
./bin/client-{arch} serve --dir . -s subdomain
```

Press Ctrl+C to stop it

## Security

If you discover any security related issues, please create an issue.

## License

The MIT License (MIT). Please see [License File](LICENSE.md) for more
information.