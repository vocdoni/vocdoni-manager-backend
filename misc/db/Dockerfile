FROM postgres:12.3
COPY ./schema.psql /
COPY ./setup.sh /
ADD ./setup.sh /docker-entrypoint-initdb.d
RUN chmod 644 /schema.psql
