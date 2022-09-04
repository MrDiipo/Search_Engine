

FROM cockroachdb/cockroach

COPY  ../linkgraph/store/cockroachdb/migrations /migrations
COPY bootstrap-db.sh .

ENTRYPOINT ["bash", "./bootstrap-db.sh"]