FROM centos:7

COPY discomon /discomon
RUN chmod +x /discomon

ADD ./dashboards/* /dashboards/

ENTRYPOINT ["/discomon"]
