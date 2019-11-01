# API Documentation

The documentation does not currently exist, so just some curls are pasted here.

## api.Rewards

```bash
curl -X POST --data-binary '{"jsonrpc": "2.0", "id": 0, "method":
"api.Rewards", "params": {"limit":20, "offset":0, "order":"", "column":""}}' \
-H 'content-type:application/json;' http://localhost:7070/api/v1
```

## api.EntrySubmissions

```bash
curl -X POST --data-binary '{"jsonrpc": "2.0", "id": 0, "method":
"api.EntrySubmissions", "params": {"limit":20, "offset":0, "order":"", "column":"", "jobid":15}}' \
-H 'content-type:application/json;' http://localhost:7070/api/v1
```

## api.SubmitSync

```bash
curl -X POST --data-binary '{"jsonrpc": "2.0", "id": 0, "method":"api.SubmitSync"}' \
-H 'content-type:application/json;' http://localhost:7070/api/v1
```