FROM centos:7

# Fix permissions so will run on OCP under "restricted" SCC
#COPY fix-permissions /usr/bin/fix-permissions
COPY discomon /discomon
RUN chmod +x /discomon

#RUN chmod +x /usr/bin/fix-permissions && \
#    /usr/bin/fix-permissions /discomon

ADD ./grafana_tpl/* /grafana_tpl/

ENTRYPOINT ["/discomon"]
