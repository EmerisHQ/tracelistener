# Local development

# **How to run Tracelistener**

1. Start cockroach DB: `cockroach demo --empty` collect the connection string (sql or sql/unix) from the console output. Keep this terminal open as long as you want the DB to live.
2. **[Optional]**Create a posix fifo: `mkfifo /tmp/tracelistener.fifo` \****if you don't create a fifo, remove the property *FIFOPath\* from the **tracelistener.toml** file and the application will make one for you. Named **./.tracelistener.fifo\*\*
3. Start tracelistener:

   1. Checkout the **_sdk-44-support_** branch.
   2. Fill **pathtorepo/build/tracelistener.toml** if you want to run the binary. Get the DB connection string from step 1.

      ```bash
      ChainName = "test"
      DatabaseConnectionURL = "postgresql://demo:demo18383@127.0.0.1:26257/defaultdb?sslmode=require"
      Type = "gaia"
      Debug = true
      FIFOPath = "/tmp/tracelistener.fifo"
      ```

4. Fill **pathtorepo/tracelistener.tom**l if you want to avail debug features of your IDE.
5. Build the binary: **make**
6. Run the binary: **pathtorepo/build/tracelistener44**
7. Init gaiad : `**~/gaiainit.sh**`

   ```bash
   #!/bin/bash

   rm -rf ~/.gaia
   # Create a key to hold your validator account
   gaiad keys add validator --keyring-backend test
   # Initialize the genesis.json file that will help you to bootstrap the network
   gaiad init test --chain-id testing
   # Add that key into the genesis.app_state.accounts array in the genesis file
   # NOTE: this command lets you set the number of coins. Make sure this account has some coins
   # with the genesis.app_state.staking.params.bond_denom denom, the default is staking
   gaiad add-genesis-account $(gaiad keys show validator --address --keyring-backend test) 1100000000stake,1000000000validatortoken
   # Generate the transaction that creates your validator
   gaiad gentx validator 1000000000stake --chain-id testing --keyring-backend test
   # Add the generated bonding transaction to the genesis file
   gaiad collect-gentxs
   ```

8. Start listening to store traces: **gaiad start –trace-store pipename [--trace]**
9. Check if balance table is updated

   ```bash
   \c tracelistener
   Select * from balances;
   ```

## **Run Test net With Multiple Validators**

1. Init the testnet and collect gentxs

   ```bash
   #!/bin/bash
   # This command will create a folder named `mytestnet` in your current directory.
   gaiad testnet --v 2 --starting-ip-address 127.0.0.1 --chain-id test_chain
   # Move to the `mytestnet` directory
   cd mytestnet
   # Collect gentexs
   gaiad collect-gentxs --gentx-dir gentxs/ --home ./node0/gaiad
   ```

2. **Change every port of node1/gaiad/config.toml and node1/gaiad/app.toml so it** does not collide with node0. [NOTE: do **not** change the **persistent_peers**]
3. Start node0 and node 1 [You should be at `mytestnet` directory at this point]

   ```bash
   #!/bin/bash
   # Start node0
   gaiad start --home ./node0/gaiad
   # Start node1 quickly from another terminal (if you delay, sometimes you'll get DB not available error)
   gaiad start --home ./node1/gaiad
   ```

### **Delegate (and re-delegate) tokens to validator**

1. Make sure you can follow **Run Test net With Multiple Validators**
2. Get validator operators `gaiad keys show node0 --bech val`

   You will see something like this

   ```bash
   name: node0
     type: local
     address: cosmosvaloper1mlr8ufu38ydnk9jjw8yp63vmh33jdjqlav7y60
     pubkey: '{"@type":"/cosmos.crypto.secp256k1.PubKey","key":"Aj4mqVXP/9/44wie6Tna98GY9FasuucQR2ou7yYAwZ/d"}'
     mnemonic: ""
   ```

   Collect the address for future use.

3. Send some coin to a 3rd party account if not done already.

   ```bash
   gaiad tx bank send node0 <address> 100stake -b block -y --chain-id test_chain --fees 2stake
   ```

4. Make the delegation

   ```bash
   gaiad tx staking delegate cosmosvaloper1mlr8ufu38ydnk9jjw8yp63vmh33jdjqlav7y60 10stake -b block -y --chain-id test_chain --from another --fees 2stake
   ```

5. For redelegation first collect the validator operator for the node1

   ```bash
   #!/bin/bash
   gaiad keys show node1 -bech val
   gaiad tx staking redelegate cosmosvaloper1mlr8ufu38ydnk9jjw8yp63vmh33jdjqlav7y60 cosmosvaloper14cuu7glq04yu4txukjycexr2ncmk76ehgnatw9 10stake -b block -y --chain-id test_chain --from another --fees 2stake
   ```
