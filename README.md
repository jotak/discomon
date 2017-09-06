# discomon

_discomon_ is a small _Go_ program, expected to run in _Kubernetes_, that fetches metrics from _Prometheus_, searches for known patterns and load corresponding dashboards in _Grafana_.

This repository contains an _OpenShift_ template to deploy _Prometheus_, _Grafana_ and _discomon_ (`prometheus-grafana-discovery.yml`).
There's another template (`wfapp/wfapp.yml`) to demo a _WildFly_ application deployed and automatically discovered in Prometheus and Grafana. This sample app is really just an empty wildfly with prometheus' JMX Exporter configured. No more.

To make _Prometheus_ discover applications, their pods must be annotated `prometheus.io/scrape: 'true'`. Cf annotation in `wfapp/wfapp.yml`. The annotation can also be added afterwards by editing YAML.

## Demo (OpenShift)

1. Create a new project and import `prometheus-grafana-discovery.yml`
2. Add `prom-discover-pods` role to `prometheus` service account (you can do it either from the web console, section _Resources / Membership_, or via command line `oc adm policy add-role-to-user prom-discover-pods -z prometheus --role-namespace=your_namespace` - just replace _your_namespace_)
3. Open Grafana: after a while you should see the Prometheus datasource and dashboard being added
4. Add to project `wfapp/wfapp.yml`
5. Check Grafana: after a while you should see a JVM dashboard being added

## Storing new dashboard templates

1. Build/edit dashboard manually as desired in Grafana
2. Get from API (not import/export), example:
    `curl -u admin:admin http://grafana-test.127.0.0.1.nip.io/api/dashboards/db/JVM` (use your grafana URL)
3. Update the json output to set dashboard id to null (that is the first "id" you should see in json)
4. Save in `dashboards/` directory
5. Rebuild docker image `./dockerbuild.sh` & push to dockerhub
