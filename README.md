# searchdump

A simple tool to backup *Search (e.g: ElasticSearch / OpenSearch) to multiple destinations

## Building

### Requirements

- Golang (v1.17+)
- Make

```bash
make build
./build/searchdump -h
```

## Running

```bash
$ searchdump -h
Usage: searchdump --from FROM --from-type FROM-TYPE --to TO --to-type TO-TYPE [--debug] [--s3-access-key S3-ACCESS-KEY] [--s3-secret-access-key S3-SECRET-ACCESS-KEY] [--s3-namespace S3-NAMESPACE] [--s3-endpoint S3-ENDPOINT] [--s3-force-path-style] [--s3-region S3-REGION]

Options:
  --from FROM, -f FROM [env: SEARCHDUMP_FROM]
  --from-type FROM-TYPE, -F FROM-TYPE
  --to TO, -t TO [env: SEARCHDUMP_TO]
  --to-type TO-TYPE, -T TO-TYPE
  --debug, -D
  --s3-access-key S3-ACCESS-KEY [env: SEARCHDUMP_S3_ACCESS_KEY]
  --s3-secret-access-key S3-SECRET-ACCESS-KEY [env: SEARCHDUMP_S3_SECRET_ACCESS_KEY]
  --s3-namespace S3-NAMESPACE [env: SEARCHDUMP_S3_NAMESPACE]
  --s3-endpoint S3-ENDPOINT [env: SEARCHDUMP_S3_ENDPOINT]
  --s3-force-path-style [env: SEARCHDUMP_S3_FORCE_PATH_STYLE]
  --s3-region S3-REGION [env: SEARCHDUMP_S3_REGION]
  --help, -h             display this help and exit
```

## Support

### Sources

- ElasticSearch (v6, v7)
- OpenSearch (v1)

### Dest

- AWS S3 compatible storage (e.g: MinIO, AWS S3)

## Project Status

Early stage - it works but it's limited.  
  
PRs are welcome.