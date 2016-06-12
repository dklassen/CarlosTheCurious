FROM postgres:9.5

COPY script/dev_postgres_setup.sh /docker-entrypoint-initdb.d/init-user-db.sh
# Adjust PostgreSQL configuration so that remote connections to the
# database are possible.
# Expose the PostgreSQL port
EXPOSE 5432

CMD ["postgres"]
