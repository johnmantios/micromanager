FROM timescale/timescaledb:latest-pg17
COPY ./../../migrations/*.up.sql /docker-entrypoint-initdb.d/
EXPOSE 5432