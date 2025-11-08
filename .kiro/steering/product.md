# Product Overview

NBU Exporter is a Prometheus exporter for Veritas NetBackup that collects and exposes backup infrastructure metrics for monitoring and visualization in Grafana.

## Core Purpose

Scrape NetBackup API endpoints to collect job statistics and storage unit metrics, exposing them in Prometheus format for time-series monitoring and alerting.

## Key Features

- **Job Metrics Collection**: Aggregates backup job statistics by type, policy, and status
- **Storage Monitoring**: Tracks storage unit capacity (free/used) for disk-based storage
- **Prometheus Integration**: Exposes metrics via HTTP endpoint for Prometheus scraping
- **Configurable Scraping**: Adjustable time windows for job data collection
- **Grafana Dashboard**: Pre-built dashboard for visualizing NetBackup statistics

## Target Users

- IT Operations teams monitoring NetBackup infrastructure
- Backup administrators tracking job success rates and storage utilization
- DevOps engineers integrating backup metrics into observability platforms

## Design Philosophy

- **API-First**: Leverages NetBackup REST API for data collection
- **Lightweight**: Single binary with minimal dependencies
- **Observable**: Structured logging and Prometheus-native metrics
- **Configurable**: YAML-based configuration for flexible deployment
