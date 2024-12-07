# get a free domain

go [here](https://myaddr.io/)

click claim your name button

choose your domain

after submit dont forget to save the private key

Update your IP Address

```bash
export PRIVATE_KEY="put your private key here"
```

replace `xxx.xxx.xxx.xxx` with your server public ip

for IPv4

```bash
wget -O- -t 1 -T 60 --post-data "key=${PRIVATE_KEY}&ip=xxx.xxx.xxx.xxx" https://ipv4.myaddr.tools/update
```

for IPv6

```bash
wget -O- -t 1 -T 60 --post-data "key=${PRIVATE_KEY}&ip=xxx.xxx.xxx.xxx" https://ipv6.myaddr.tools/update
```

## Things to Know

- You must update each IP address (IPv4 and IPv6) at least once every 90 days
  for that IP to remain active.
- IPv4 and IPv6 updates are tracked independently.
- If no updates are made to your name for 120 days, your registration is deleted
  and your name released.
