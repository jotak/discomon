# discomon

_discomon_ is a small _Go_ program, expected to run in _Kubernetes_, that fetches metrics from _Prometheus_, searches for known patterns and load corresponding dashboards in _Grafana_.

This repository contains an _OpenShift_ template to deploy _Prometheus_, _Grafana_ and _discomon_ (`prometheus-grafana-discovery.yml`).
There's another template (`example/wfapp/wfapp.yml`) to demo a _WildFly_ application deployed and automatically discovered in Prometheus and Grafana. This sample app is really just an empty wildfly with prometheus' JMX Exporter configured. No more.

To prevent _Prometheus_ from discovering an application, its pod must be annotated `prometheus.io/scrape: 'false'` ([like this](https://github.com/jotak/discomon/blob/6c098e27c4cae41021b2551251a6e8e659134f1a/prometheus-grafana-discovery.yml#L163-L164)).

## OpenShift demos

### Demo 1: an empty WildFly app

1. Create a new project and import `prometheus-grafana-discovery.yml`
2. Open Grafana: after a while you should see the Prometheus datasource and dashboard being added
3. Add to project `examples/wfapp.yml`
4. Check Grafana: after a while you should see a JVM dashboard being added

### Demo 2: scenario with OpenTracing

1. Repeat steps 1 and 2 of demo 1 to setup discomon in OpenShift, or use the existing setup
2. [Optionally] add Jaeger into the project: `oc process -f https://raw.githubusercontent.com/jaegertracing/jaeger-openshift/master/all-in-one/jaeger-all-in-one-template.yml | oc create -f -`
3. Run `oc create -f examples/otapp.yml` (these demo microservices come from https://github.com/objectiser/opentracing-prometheus-example)
4. Now when you hit a URL that involves OpenTracing (for instance: `http://ordermgr-test.127.0.0.1.nip.io/buy`), metrics will be created in Prometheus, and based on that discomon will create the OpenTracing dashboard in Grafana after a few seconds.

### Demo 3: a Vert.X game with application metrics

1. Repeat steps 1 and 2 of demo 1 to setup discomon in OpenShift, or use the existing setup
2. Click on _Add to Project_ > _Deploy Image_ and provide image name `jotak/falco-the-hawk:prometheus`. Click on _Create_.
In Grafana you will soon see 3 dashboards: the usual _Prometheus_, _JVM_ which is there because the supplied docker image comes with Prometheus _JMX Exporter_, and _Vert.X_.
3. Now, there's a dashboard for my game metrics that I would like to use. I could manually import it to Grafana, but it may not be the recommended way to go, think _immutable_. I'd rather add it as a _ConfigMap_. So click on _Resources_ > _Config Maps_ > _discomon-config_ > _Edit_.
4. Because the game metrics are prefixed with `falco`, we will add a pattern in _config.yml_:

```yml
  - patterns: ["^falco.*"]
    name: "falco"
    category: "game"
```
5. Click on _Add Item_, enter key `falco.json` and in _Value_ paste the content of this file: https://raw.githubusercontent.com/jotak/falco-demo/prometheus/docker-graf/Falco.json . Save.
6. In order to have game metrics, you must play a little bit. Create a route on falco-the-hawk service (default parameters), and play!
7. :-( At this point, nothing happens because discomon didn't reload its config files. We have to kill the pod, it will automatically restart and push our dashboard to _Grafana_. You can also check again the Vert.X dashboard, it has started to animate a bit more.


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
