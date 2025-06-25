# 1. One‑time server bootstrap
`./server_setup.sh ubuntu@203.0.113.5 example.com`

# 2. Any time you have new code
`./deploy.sh ubuntu@203.0.113.5`

By default deploy.sh prunes old releases after deployment. 

Set `PRUNE=false` to keep all past releases.

`PRUNE=false ./deploy.sh ubuntu@203.0.113.5`

