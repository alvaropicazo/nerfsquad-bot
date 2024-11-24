# solana-bot
Solana-bot tracker which sends transactions autonomously written in Golang


## How does it work

**Solana-bot** is a program written in Golang designed to track a chosen wallet 24/7, specified through a configuration file passed to the program. It continuously monitors for new transactions to replicate, and thanks to Go's threading capabilities, it formats them for our specific use case and sends them to the selected DEX, all while continuing to track the wallet.

## Configuration Files

- <code>apis.yaml</code>: Contains Endpoints to obtain information about the wallet to track and the DEX to which transactions will be sent.
- <code>keys.yaml</code>: API Keys.
- <code>wallets.yaml</code>: It contains the addresses of the wallets of the different DEXs with which the wallet we want to track interacts, and <code>external_wallet</code> is the wallet selected to replicate the transactions.
- <code>cryptokeys.yaml</code>: It contains the addresses of the wallets of the different DEXs with which the wallet we want to track interacts, and <code>external_wallet</code> is the wallet selected to replicate the transactions.

## How to run solana-bot

### PRE-REQUISITES

- Have go installed
- **IMPORTANT** Create a cryptokeys.yaml in config folder which contains your wallets' crypto material to send transactions to the DEX

<code>go run .</code>