# 1. One‑time server bootstrap
`./server_setup.sh ubuntu@203.0.113.5 example.com`

# 2. Any time you have new code
`./deploy.sh ubuntu@203.0.113.5`

By default deploy.sh prunes old releases after deployment. 

Set `PRUNE=false` to keep all past releases.

`PRUNE=false ./deploy.sh ubuntu@203.0.113.5`

## Litestream backups

`server_setup.sh` installs [Litestream](https://litestream.io) to stream the
`app.db` SQLite database to Digital Ocean Spaces. Credentials can be provided
via `ACCESS_KEY_ID` and `SECRET_ACCESS_KEY` environment variables (or as the
third and fourth command line arguments).

To restore the latest backup locally:

```bash
litestream restore -config /etc/litestream.yml -o app.db
```

This downloads the most recent snapshot from your Space and writes it to
`app.db` in the current directory.

