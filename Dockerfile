FROM centos:7

COPY discomon /discomon
RUN chmod +x /discomon

COPY assets /assets
ADD ./dashboards/* /dashboards/

ENTRYPOINT ["/discomon"]
