#!/usr/bin/env bash


sonar-scanner \
  -Dsonar.projectKey=golang-nbu-exporter \
  -Dsonar.sources=. \
  -Dsonar.host.url=https://prod-sqube-srv1 \
  -Dsonar.login=a577749bb28455d35463e716531a0354c1d7bea2