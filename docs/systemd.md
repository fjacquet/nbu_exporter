# systemd (EL9 host)

Docker is **not** required — the exporter is a single static (`CGO_ENABLED=0`) binary. For a
non-container deployment on Enterprise Linux 9, use the unit shipped in `deploy/`.

## Install

```bash
# user + binary
sudo useradd --system --no-create-home --shell /usr/sbin/nologin nbu
sudo install -m 0755 bin/nbu_exporter /usr/local/bin/nbu_exporter

# config + secrets
sudo install -d -o root -g nbu -m 0750 /etc/nbu_exporter
sudo install -m 0640 -o root -g nbu config.yaml /etc/nbu_exporter/config.yaml
sudo install -m 0600 -o root -g nbu deploy/nbu_exporter.env.example /etc/nbu_exporter/nbu_exporter.env
# edit /etc/nbu_exporter/nbu_exporter.env to set NBU1_APIKEY=...

# service
sudo install -m 0644 deploy/nbu_exporter.service /etc/systemd/system/nbu_exporter.service
sudo systemctl daemon-reload
sudo systemctl enable --now nbu_exporter
```

Set `logName: ""` in `config.yaml` so logs go to the journal.

## Operate

```bash
journalctl -u nbu_exporter -f         # follow logs
sudo systemctl reload nbu_exporter    # live config reload (sends SIGHUP)
sudo systemctl status nbu_exporter
```

## Hardening

The unit runs as the unprivileged `nbu` user inside a sandbox:

- `NoNewPrivileges=true`, `ProtectSystem=strict`, `ProtectHome=true`
- `PrivateTmp`, `PrivateDevices`, `ProtectKernel*`, `ProtectControlGroups`
- `RestrictAddressFamilies=AF_INET AF_INET6`, `RestrictNamespaces`, `LockPersonality`
- `Restart=on-failure`

Secrets are supplied through the `EnvironmentFile` and referenced as `${NBU1_APIKEY}`
in `config.yaml`. Keep that file mode `0600`.

## macOS (launchd / Homebrew)

On macOS run it under **launchd** (the systemd equivalent). `brew services` is not wired up:
the Homebrew cask only installs the binary on your PATH — it defines no service block — so
register a `launchd` job yourself, e.g. `~/Library/LaunchAgents/com.fjacquet.nbu_exporter.plist`
with `ProgramArguments` `[/opt/homebrew/bin/nbu_exporter, --config, <path>/config.yaml]` and
`RunAtLoad`/`KeepAlive` set, then `launchctl load` it.
