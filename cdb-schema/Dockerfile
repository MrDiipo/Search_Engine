

FROM cockroachdb/cockroach

COPY inkgraph/store/cockroachdb/migrations /migrations
COPY Chapter10/cdb-schema/bootstrap-db.sh .

ENTRYPOINT ["bash", "./bootstrap-db.sh"]