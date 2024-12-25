# Nerf Squad Bot
Solana-bot tracker which sends transactions autonomously written in Golang.


## How does it work

**NS Bot** is a program written in Golang designed to track a chosen wallet 24/7, specified through a configuration file passed to the program. It continuously monitors for new transactions to replicate, and thanks to Go's threading capabilities, it formats them for our specific use case and sends them to the selected DEX, all while continuing to track the wallet.

Due to Golang Raydium SDK limitations (it's not an official implementation of the library), we had to create a small service written in Javascript which sends the transactions to the mainnet.

The application sends to a Telegram Group Chat every successful transactions' details.
## Configuration Files

- <code>apis.yaml</code>: Contains Endpoints to obtain information about the wallet to track and the DEX to which transactions will be sent.
- <code>keys.yaml</code>: API Keys.
- <code>wallets.yaml</code>: It contains the addresses of the wallets of the different DEXs. <code>external_wallet</code> is the wallet selected to replicate the transactions.


```
dex:
    - xxxxxxxxxx
external_wallet: xxxxxxxx
```

- <code>cryptokeys.yaml</code>: It contains the addresses of our own wallets, and <code>personal_wallet</code> is the wallet selected to replicate the transactions.

```
personal_wallet: xxxxxxxx
```

## How to run solana-bot

### PRE-REQUISITES

- Have go 1.23 installed
- Set PRIV_KEY env var in a .env file under /send_transactions_app folder.
- Have node installed
- Have typescript installed
- Run <code>npm install</code> in the send_transactions_app folder

- **IMPORTANT:** Create all previous configuration files under the config folder.


<code>npm run dev</code> in the /send_transactions_app folder

<code>go run .</code> in ~/