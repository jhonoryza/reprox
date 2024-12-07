# generate letsencrypt certificate

1. install
   [docker engine](https://docs.docker.com/engine/install/ubuntu/#install-using-the-convenience-script)

2. create a required directory

```bash
mkdir -p ~/letsencrypt/certs ~/letsencrypt/config ~/letsencrypt/workdir
```

3. generate a free certificate change `labstack.myaddr.io` with your own domain.

```bash
docker run -it --rm \
  -p 80:80 \ 
  -v ~/letsencrypt/certs:/etc/letsencrypt \
  -v ~/letsencrypt/config:/config \
  -v ~/letsencrypt/workdir:/workdir \
  certbot/certbot certonly --standalone \
  -d labstack.myaddr.io
```
