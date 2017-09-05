# discomon

_discomon_ is a small _Go_ program, expected to run in _Kubernetes_, that fetches metrics from _Prometheus_, searches for known patterns and load corresponding dashboards in _Grafana_.

This repository contains an _OpenShift_ template to deploy _Prometheus_, _Grafana_ and _discomon_ (`prometheus-grafana-discovery.yml`).
There's another template (`wfapp/wfapp.yml`) to demo a _WildFly_ application deployed and automatically discovered in Prometheus and Grafana.

To make _Prometheus_ discover applications, their pods must be annotated `prometheus.io/scrape: 'true'`. Cf annotation in `wfapp/wfapp.yml`. The annotation can also be added afterwards by editing YAML.

## Demo (OpenShift)

_Prerequisites:_ unfortunately it is currently necessary to have the `cluster-reader` role in OpenShift / Kubernetes. Prometheus cannot discover pods otherwise. To do so run `oc adm policy add-cluster-role-to-user cluster-reader -z default` while you're logged in as admin under the project/namespace you want to use.

1. Create a new project and import `prometheus-grafana-discovery.yml`
2. Open Grafana: after a while you should see the Prometheus datasource being added
3. Add to project `wfapp/wfapp.yml`
4. Check Grafana: after a while you should see a JVM dashboard being added
