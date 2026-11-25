# laser-mtproxy

English | [Русский](README.ru.md)

An MTProto proxy for Telegram applications and a Bot API gateway for bots built on `laser_tele`. The server runs on a
separate machine in the local network and is managed through a web interface. It's a Go port of
[tg-ws-proxy](https://github.com/Flowseal/tg-ws-proxy) with the same routes to Telegram; the web interface replaces
the tray application.

```
Telegram Desktop / phone ──MTProto (dd/ee)──► :1443 ─┬─► wss://kwsN.web.telegram.org (pool, DC → IP)
                                                     ├─► CF Worker ─► datacenter IP
                                                     ├─► CF domains: WebSocket, media over HTTP/2
                                                     └─► TCP to the datacenter IP
laser_tele bot ──HTTP CONNECT / SOCKS5──► :1080 ──► api.telegram.org:443 (DNS, fallback IPs, CF Worker)
administrator ──browser──► :8080 (web interface)
```

Bots talk to the Bot API over HTTPS, so an MTProto proxy can't carry their traffic. For them the server runs an
HTTP CONNECT and SOCKS5 proxy on a separate port. It opens tunnels only to the allowed destinations
(`api.telegram.org:443`). TLS stays end to end: the gateway sees only encrypted bytes.

## Running

You need Go 1.24 or newer. The only external dependency is `github.com/boombuler/barcode` for QR codes.

```
cd mtproxy
go build -o laser-mtproxy .
cp dist/laser-mtproxy.json .
./laser-mtproxy -config laser-mtproxy.json
```

On the first start the server generates the users' secrets, writes them to the configuration and prints the
`tg://proxy?...` link of the first user. The web interface is at `http://<machine IP>:8080`. If the configuration
file doesn't exist, the server creates it with default values.

`./build.sh` builds for Linux, Windows and macOS into `build/`.

Flags: `-config path`, `-v` (debug log), `-check` (check the configuration without changing it), `-version`,
`--install`, `--uninstall`, `--purge`.

## Installing as a systemd service

```
./laser-mtproxy --install                          # default configuration
./laser-mtproxy --install -config my-config.json   # initial configuration
```

| Run by | Service | Directory (binary, configuration, logs, statistics) | Log |
|---|---|---|---|
| a regular user | user service, `~/.config/systemd/user/laser-mtproxy.service` | `~/.local/share/laser-mtproxy` | `journalctl --user -u laser-mtproxy` |
| root | system service, `/etc/systemd/system/laser-mtproxy.service`, runs as the system user `laser-mtproxy` | `/opt/laser-mtproxy` | `journalctl -u laser-mtproxy` |

Running `--install` again updates the binary and restarts the service. An existing configuration is kept;
`-config` matters only for the first installation. A user service has to keep running after you log out, so the
installer turns on lingering (`loginctl enable-linger`); if it lacks the rights, it prints the command to run as
root. A system service can listen on ports below 1024, for example 443 for Fake TLS.

`--uninstall` stops and removes the service and keeps the configuration with secrets, the logs and the statistics.
`--uninstall --purge` removes them too, and for a system service also the `laser-mtproxy` user.

## Access and passwords

The shipped configuration trusts this machine and its local network:

```json
"access": {
  "trusted_networks": ["loopback", "lan"],
  "username": "admin"
}
```

`lan` means the private subnets of the machine's network interfaces (10/8, 172.16/12, 192.168/16, 100.64/10,
fc00::/7, link-local). The server recalculates them every minute. The public subnet of a server in a datacenter isn't
part of `lan`. The list also takes `private` (all private addresses), `any` (every address), CIDR ranges
(`10.0.0.0/8`) and single IPs.

| Entry point | Trusted address | Other addresses |
|---|---|---|
| Web interface | no password | closed (`web.admin_trusted_only`, on by default); with it off, username and password |
| Personal pages `/p/<token>` | open | open (the token in the address is the key) |
| Bot API gateway | no password | `http://admin:password@host:1080` or SOCKS5 with username and password |
| MTProto | all enabled users | only users with `"allow_external": true` |

MTProto has no password: the client encrypts the connection with the user's secret. Until a password is set, the web
interface and the gateway refuse untrusted addresses. Set the password in the web interface or as `"password": "..."`
inside `access`; at startup the server replaces it with the `password_hash`.

If a user without `allow_external` connects from an untrusted address, the server refuses the connection, logs a
warning and counts it in the report as "outside".

## Users

Each user has a secret and a limit of simultaneous connections:

```json
"users": [
  {"name": "lan",   "secret": "", "max_connections": 100, "allow_external": false, "comment": "shared secret for the local network"},
  {"name": "ivan",  "secret": "", "max_connections": 12,  "allow_external": true},
  {"name": "guest", "secret": "", "max_connections": 0,   "allow_external": false}
]
```

- `max_connections`: `0` disables the user, `N` allows up to N simultaneous connections. A connection over the limit
  is closed and counted in the report as "limit".
- One Telegram application keeps several connections: the main one and more for media downloads. Leave room, for
  example 10 to 20 per device, and check the peak in the report.
- The server fills an empty `secret` with a random one at startup or on saving.
- `allow_external` lets the user in from outside the trusted networks.

The Users tab is the easier way to manage users: add, change the limit, disable, generate a new secret, delete.
Changes take effect at once without restarting the proxy. Disabling or deleting a user, or giving them a new secret,
closes their open connections; lowering the limit doesn't.

The secret is the user: whoever has the link connects as that user. The server can't tell a forwarded link from the
original, but the report shows the signs (several addresses at once, connections hitting the limit). Then give the
user a new secret.

A version 1.0 configuration with `secret` and `external_secret` is converted at startup: they become the users `lan`
and `external`, and the old links keep working.

### Personal page with a QR code

The "QR and link" button opens `http://<server>:8080/p/<token>`: a QR code, an "Open in Telegram" button, the link
to copy, the code as a PNG and the parameters for manual setup. The "Page address" button copies the address of this
page, so you can send it to the person. The page needs no password; the token comes from the secret and can't be
guessed. After "New secret" the old page stops working.

Links of users with `allow_external` contain `public_host` when it's set, otherwise the address the page was opened
with. Set `public_host` (the external IP or name) for external users.

### Report

The Report tab shows for each user: connections now and the limit, the peak and when it happened, the number of
connections, refusals (limit, disabled, outside), traffic, the last connection with its address, and the addresses of
current connections. The counters are saved every minute and on shutdown to `laser-mtproxy-stats.json` next to the
configuration. The report can be downloaded as CSV (it opens in Excel) and reset.

### Proxy open to the internet, administration only from inside

1. `host: "0.0.0.0"`: MTProto listens on all addresses, the public one included.
2. `public_host`: the public address or name of the server; it goes into the links of external users.
3. Users who need access from outside get `allow_external: true`; their secret works as the password. Everyone else
   is refused from outside.
4. `web.host`: a local address, so the administration doesn't listen on the public one. With
   `web.admin_trusted_only: true` the administration doesn't answer untrusted addresses even if the web port is
   opened to the internet; only personal pages work there.
5. The Bot API gateway doesn't have to be open from outside: set `gateway.host` to a local address.

From a trusted address the web interface opens only by IP, `localhost` or the name of this machine. Add other names
to `web.allowed_hosts`. This blocks DNS rebinding: a foreign website can't control the proxy through the browser of
someone in the local network.

## Connecting Telegram

Send the person the address of their personal page or show the QR code. On a phone, the simplest way is to open the
page and tap "Open in Telegram". Manually: Settings → Advanced → Connection type → Proxy → MTProto, with the server,
port and secret from the page.

If photos and videos don't load, leave only `4:149.154.167.220` in "DC → IP" or clear the field.

## Connecting a bot

```go
laser_tele.DoLaserTeleInit(laser_tele.LaserTeleConfigT{
	Proxy: "http://192.168.1.10:1080", // or socks5://192.168.1.10:1080
})
```

or the environment variable `TG_PROXY=http://192.168.1.10:1080`. From an untrusted address use
`http://admin:password@192.168.1.10:1080`.

The gateway connects to the DNS address of `api.telegram.org`. After 1.5 seconds it also tries the fallback IPs from
`gateway.fallback_ips` and remembers the address that worked. Telegram frontends (`149.154.167.220` and others) serve
the Bot API with a valid `api.telegram.org` certificate, so this works where the address of `api.telegram.org` is
blocked. The last fallback is a CF Worker (`gateway.via_cfproxy_worker`).

## Checking

`mtprobe` connects to the proxy as a Telegram client, sends `req_pq_multi` to the datacenters and waits for `resPQ`
from Telegram:

```
go run ./cmd/mtprobe -addr 192.168.1.10:1443 -secret dd<secret> -dc 1,2,3,4,5,-2,-4
DC1        OK    resPQ in 532 ms
DC2        OK    resPQ in 45 ms
...
```

A negative number is a media DC. For an ee-secret pass it whole; `mtprobe` goes through Fake TLS. To check the
gateway, run `curl -x http://192.168.1.10:1080 https://api.telegram.org/bot123:x/getMe`: the answer
`{"ok":false,"error_code":401,...}` comes from the real Bot API.

The Tests tab of the web interface checks the direct DC → IP route, the CF domains and the CF Worker.

## Configuration

The field names of the MTProto part are the same as in tg-ws-proxy.

| Field | Default | Meaning |
|---|---|---|
| `host`, `port` | `0.0.0.0`, `1443` | address of the MTProto proxy |
| `public_host` | empty | address in the links of external users; empty means the address the web interface was opened with |
| `users` | `lan` with 100 connections | users: `name`, `secret`, `max_connections`, `allow_external`, `comment` |
| `dc_ip` | `2:149.154.167.220`, `4:149.154.167.220` | direct WebSocket route to a DC |
| `pool_size`, `buf_kb` | `4`, `256` | ready WS connections per DC; socket buffer |
| `cfproxy`, `h2` | `true`, `true` | fallback through CF domains; media over HTTP/2 |
| `cfproxy_user_domain_enabled`, `cfproxy_user_domain` | off | own CF domains instead of the built-in ones |
| `cfproxy_domains_url` | the tg-ws-proxy list on GitHub | CF domain list refreshed every hour; empty means the built-in list only |
| `cfproxy_worker_enabled`, `cfproxy_worker_domain` | off | Cloudflare Worker domains |
| `no_secure` | `false` | port 80 without TLS for the CF proxy and the Worker |
| `force_test_dc` | `false` | all traffic to the test DCs |
| `fake_tls_domain` | empty | Fake TLS (ee-secret) and masking as this site |
| `proxy_protocol` | `false` | PROXY v1 header from nginx/haproxy (accepted only from a trusted address) |
| `verbose`, `log_file`, `log_max_mb` | `false`, `laser-mtproxy.log`, `5` | log; the path is relative to the configuration |
| `access.*` | see above | trusted networks, username, password |
| `gateway.enabled`, `host`, `port` | `true`, `0.0.0.0`, `1080` | Bot API gateway |
| `gateway.allowed_hosts` | `api.telegram.org:443` | where the gateway opens tunnels: `host:port`, `*.domain:port`, port `*` |
| `gateway.fallback_ips` | `api.telegram.org: 149.154.167.220` | fallback addresses of hosts |
| `gateway.via_cfproxy_worker` | `true` | last fallback of the gateway through a CF Worker |
| `web.host`, `web.port` | `0.0.0.0`, `8080` | web interface (applied after the process restarts) |
| `web.tls_cert`, `web.tls_key` | empty | HTTPS for the web interface |
| `web.language` | `ru` | default language |
| `web.allowed_hosts` | empty | extra host names of the web interface |
| `web.admin_trusted_only` | `true` | administration only from trusted networks; no password login from outside |

The tg-ws-proxy documentation describes how to set up a Cloudflare Worker and your own CF domain
(`docs/EN/CfWorker.md`, `docs/EN/CfProxy.md`); the Worker code is the same.

## Differences from tg-ws-proxy

- A web interface instead of the tray and the settings window: links, restart, settings, CF tests, statistics, logs.
- Trusted networks and a password.
- Users with their own secrets and connection limits, personal pages with QR codes, a report.
- A Bot API gateway for bots.
- No update check and no Windows autostart; there's a systemd unit for Linux instead.
