FROM chromedp/headless-shell:stable

ARG ARCH="amd64"
ARG OS="linux"
RUN apt update && apt install dumb-init
COPY blackbox_exporter  /bin/blackbox_exporter
COPY blackbox.yml       /etc/blackbox_exporter/config.yml

EXPOSE      9115
ENTRYPOINT ["dumb-init", "--", "/bin/blackbox_exporter"]
CMD         ["--config.file=/etc/blackbox_exporter/config.yml"]
