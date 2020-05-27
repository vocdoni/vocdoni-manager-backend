FROM postgres:latest
COPY ./schema.psql /
ADD ./setup.sh /docker-entrypoint-initdb.d
RUN chmod 644 /schema.psql
