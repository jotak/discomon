# discomon

_discomon_ is a small _Go_ program, expected to run in _Kubernetes_, that fetches metrics from _Prometheus_, searches for known patterns and load corresponding dashboards in _Grafana_.

This repository contains an _OpenShift_ template to deploy _Prometheus_, _Grafana_ and _discomon_ (`prometheus-grafana-discovery.yml`).
There's another template (`example/wfapp/wfapp.yml`) to demo a _WildFly_ application deployed and automatically discovered in Prometheus and Grafana. This sample app is really just an empty wildfly with prometheus' JMX Exporter configured. No more.

To prevent _Prometheus_ from discovering an application, its pod must be annotated `prometheus.io/scrape: 'false'` ([like this](https://github.com/jotak/discomon/blob/6c098e27c4cae41021b2551251a6e8e659134f1a/prometheus-grafana-discovery.yml#L163-L164)).

## Demo (OpenShift)

1. Create a new project and import `prometheus-grafana-discovery.yml`
2. Add `prom-discover-pods` role to `prometheus` service account (you can do it either from the web console, section _Resources / Membership_, or via command line `oc adm policy add-role-to-user prom-discover-pods -z prometheus --role-namespace=your_namespace` - just replace _your_namespace_)
3. Open Grafana: after a while you should see the Prometheus datasource and dashboard being added
4. Add to project `examples/wfapp.yml`
5. Check Grafana: after a while you should see a JVM dashboard being added

### Scenario with OpenTracing

1. Repeat steps 1 to 3 of OpenShift demo to setup discomon in OpenShift, or use the existing setup
2. [Optionally] add Jaeger into the project: `oc process -f https://raw.githubusercontent.com/jaegertracing/jaeger-openshift/master/all-in-one/jaeger-all-in-one-template.yml | oc create -f -`
3. Run `oc create -f examples/otapp.yml` (these demo microservices come from https://github.com/objectiser/opentracing-prometheus-example)
4. Now when you hit a URL that involves OpenTracing (for instance: `http://ordermgr-test.127.0.0.1.nip.io/buy`), metrics will be created in Prometheus, and based on that discomon will create the OpenTracing dashboard in Grafana after a few seconds.

## Storing new dashboard templates

1. Build/edit dashboard manually as desired in Grafana
2. Get from API (not import/export), example:
    `curl -u admin:admin http://grafana-test.127.0.0.1.nip.io/api/dashboards/db/JVM` (use your grafana URL)
3. Update the json output to set dashboard id to null (that is the first "id" you should see in json)
4. Save in `dashboards/` directory
5. Rebuild docker image `./dockerbuild.sh` & push to dockerhub

## Dev how-to

For development, use the OpenShift template `prometheus-grafana-discovery-dev.yml` instead of the other one.

Once imported in OpenShift, `discomon` won't run because it expects a build. From the command line run:

```bash
go build; oc start-build discomon --from-dir=. --follow
```

Repeat this command every time you want to update the deployment.
