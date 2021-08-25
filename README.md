# nbu_exporter
Netbackup exporter for prometheus + Grafana dashboard

## Debug
To debud you need to install  Delve, this command should work:
```bash
$ go install github.com/go-delve/delve/cmd/dlv@latest
```

## Run

You can call as simple as
```bash
./nbu_exporter config config.yaml
```

But you need to
- create an api key in NBU UI
- configure config.yaml file


## Grafana

One scrapped by prometheus, you can load the json in grafana folder to your system