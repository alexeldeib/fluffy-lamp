FROM debian:buster

RUN apt update && apt install -y curl gnupg2

RUN curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc > /tmp/pgsql.gpg
RUN apt-key add /tmp/pgsql.gpg
RUN echo "deb http://apt.postgresql.org/pub/repos/apt/ buster-pgdg main" > /etc/apt/sources.list.d/pgdg.list
RUN apt-get update && apt-get install -y postgresql-client-12

RUN apt update && apt install -y python3 python3-pip python3-virtualenv python3-venv gettext-base

ENV VIRTUAL_ENV=/opt/venv
RUN python3 -m virtualenv --python=/usr/bin/python3 ${VIRTUAL_ENV}
ENV PATH "$PATH:${VIRTUAL_ENV}/bin"
RUN pip install --upgrade pip setuptools pipx
RUN pip install tusker

RUN apt install -y locales 
RUN locale-gen en_US.UTF-8

ENV LC_ALL=C.UTF-8
ENV LANG=C.UTF-8

COPY entrypoint.sh entrypoint.sh
RUN chmod +x entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
