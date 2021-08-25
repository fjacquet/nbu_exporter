# Prometheus Netbackup exporter
Netbackup exporter for prometheus + Grafana dashboard

![Code Analysis](https://github.com/fjacquet/nbu_exporter/actions/workflows/codeql-analysis.yml/badge.svg)
![Go build](https://github.com/fjacquet/nbu_exporter/actions/workflows/go.yml/badge.svg)


## Run

You can call as simple as
```bash
./nbu_exporter config config.yaml
```

But you need to
- create an api key in NBU UI
- configure config.yaml file


## Grafana dashboard

One scrapped by prometheus, you can load the json in grafana folder to your system


## Debug
To debug, you need to install  Delve, this command should work:
```bash
$ go install github.com/go-delve/delve/cmd/dlv@latest
```
