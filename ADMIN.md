# Admin Documentation

The pool binary also has some admin level cli functions.

## Pool CLI

### Make a user an Admin

To see admin pages, you can promote a user to an admin:

```bash
prosper-pool db admin user@gmail.com
```

### To make a new invite code

Users need an invite code to join the pool

```bash
prosper-pool db code
```

### To constuct the payments json for submission

To payout your users, you need to construct the payment json. A secondard cli will submit this payment object to the network, then it will save a new payment json to disk. This final payment json can then be submitted back to the pool to record the payment in the database

```bash
prosper-pool db payout payments.json
```

### To record the paid payouts

Once the payout is recorded, and you verfied it worked on the pegnet network, you can record the payment on the pool.

```bash
prosper-pool db record receipt.json
```

## Payout-CLI

The payout CLI needs acces to a factom-walletd and a factomd to create and submit the transaction.

### Submit a payment json object

The EC address must have some ecs and the FA address must have enough PEG to cover the transaction. The receipt is saved to the receipt filepath. You should keep these json documents.

```
payout-cli pay payments.json FA2jK2HcLnRdS94dEcU27rF3meoJfpUcZPSinpb7AwQvPRY6RL1Q EC3TsJHUs8bzbbVnratBafub6toRYdgzgbR7kWwCW4tqbmyySRmg receipt.json

```
