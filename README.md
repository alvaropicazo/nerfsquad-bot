# solana-bot
Solana-bot tracker which sends transactions autonomously written in Golang.


## How does it work

**Solana-bot** is a program written in Golang designed to track a chosen wallet 24/7, specified through a configuration file passed to the program. It continuously monitors for new transactions to replicate, and thanks to Go's threading capabilities, it formats them for our specific use case and sends them to the selected DEX, all while continuing to track the wallet.

Due to Golang Raydium SDK limitations (it's not an official implementation of the library), we had to create a small service written in Javascript which sends the transactions to the mainnet.

The application sends to a Telegram Group Chat every successful transactions' details.
## Configuration Files

- <code>apis.yaml</code>: Contains Endpoints to obtain information about the wallet to track and the DEX to which transactions will be sent.
- <code>keys.yaml</code>: API Keys.
- <code>wallets.yaml</code>: It contains the addresses of the wallets of the different DEXs with which the wallet we want to track interacts, and <code>external_wallet</code> is the wallet selected to replicate the transactions.
- <code>cryptokeys.yaml</code>: It contains the addresses of the wallets of the different DEXs with which the wallet we want to track interacts, and <code>external_wallet</code> is the wallet selected to replicate the transactions.

## How to run solana-bot

### PRE-REQUISITES

- Have go installed
- Set PRIV_KEY env var.
- Have node installed
- Run <code>npm install</code> in the send_transactions_app folder

- **IMPORTANT** Create all configuration files under the config folder.


<code>npm run dev</code>

<code>go run .</code>