# Admin Documentation

The pool binary also has some admin level cli functions.

## Pool CLI

The pool cli requires access to the pool binary. If you are running with the docker compose, you can do this:

```bash
docker exec -it prosper-pool /bin/bash
echo "The --phost is the postgres host, which is '$DB' from within the container"
/go/bin/prosper-pool --phost $DB
```

### Make a user an Admin

To see admin pages, you can promote a user to an admin. A user cannot be demoted by the cli at this time, so use with caution.

```bash
prosper-pool db admin user@gmail.com
```

### To make a new invite code

Users need an invite code to join the pool. A single invite code is created and can only be redeemed **once**. Once the code is claimed by a user, that code cannot be used again.

```bash
prosper-pool db code
```

### To construct the payments json for submission

__Step 1__ to paying out users in the pool

To payout your users, you need to construct the payment json. A secondardy cli will submit this payment object to the network, then it will save a new payment json to disk. This final payment json can then be submitted back to the pool to record the payment in the database

```bash
prosper-pool db payout payments.json
```

### To record the paid payouts

__Step 3__ to paying out users in the pool

Once the payout is recorded, and you verfied it worked on the pegnet network, you can record the payment on the pool. This will update your postgres database with records recording the payment.

```bash
prosper-pool db record receipt.json
```

## Payout-CLI

The payout CLI needs acces to a factom-walletd and a factomd to create and submit the transaction.

### Submit a payment json object

__Step 2__ to paying out users in the pool

To submit the `payments.json` to the peg network, you use the `payout-cli`. This is so the private keys can be kept on a different machine as the pool. The `payout-cli` will read a `payments.json` file, make the batch transaction to pay the users in your pool, and create a `receipt.json`. This receipt should be recorded by the pool once the tx is verfied to be completed and valid.

The EC address must have some ecs and the FA address must have enough PEG to cover the transaction. The receipt is saved to the receipt filepath that you specified. You should keep these json documents.

```
payout-cli pay payments.json FA2jK2HcLnRdS94dEcU27rF3meoJfpUcZPSinpb7AwQvPRY6RL1Q EC3TsJHUs8bzbbVnratBafub6toRYdgzgbR7kWwCW4tqbmyySRmg receipt.json


# To ensure the payout worked, wait for the block to complete, then
pegnetd get tx <entry-hash>

# If the result is the transaction body in json, then the tx was executed by pegnet.
# If you get back :
#    'jsonrpc2.Error{Code:-32803, Message:"Transaction Not Found", Data:"no matching tx-id was found"}'
# Then the tx could have been rejected by pegnet, or the entry did not make
# it into the blockchain.

```
