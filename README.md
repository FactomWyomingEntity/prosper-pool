# Prosper Pool

Prosper is a mining pool implementation for PegNet and works with `prosper-miner`. The pool handles polling datasources and syncing the pegnet chain to come up with the oprhash for miners to mine. When a miner submits their work to the pool in the form of a share, the pool software determines whether or not to submit that share, and will credit the miner for the work they have done.

# Design

There is a few moving parts in the pool.

![image](imgs/pool.png)

## Notes

### Rolling Submissions

The pegnet reference miner requires a node that syncs with minutes. If the minute syncing is lost, the miner is dead in the water. Prosper pool does not require syncing by minutes, and uses a rolling submission strategy. If your hashpower begins to dominate the network, tweaking might be necessary. A 36 block (6 hr) exponential moving average is kept of the network difficulty to determine whether or not to submit a share.

### Payouts

What we owe miners is recorded, but no payouts actually occur. This is to be implemented at a future date.

### Stopping the pool

All miner work is stored in memory and saved to postgres at the start of the next block. If the pool is shut down, the miner work for that block is lost and the pool will receive the full payout.


# Development enviroment

## Postgres instance

```
echo "launch the postgres db"
docker-compose up -d
```

## User Authentication

I say we look at something like this for user management: https://github.com/qor/auth