FROM postgres:latest
COPY ./schema.psql /root
ADD ./setup.sh /docker-entrypoint-initdb.d
