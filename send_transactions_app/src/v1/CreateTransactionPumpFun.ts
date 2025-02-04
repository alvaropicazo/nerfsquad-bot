import {
    Connection,
    PublicKey,
    VersionedTransaction,
    Keypair,
} from '@solana/web3.js'
import yaml from 'js-yaml'
import bs58 from 'bs58';
import { Request, Response } from 'express'
import fs from 'fs'
import dotenv from 'dotenv'

dotenv.config()

type KeyObj = {
    personal_wallet: string
} | undefined

type TokenObj = {
    keys: Keys
} | undefined

type Keys = {
    instantnodes: string
    coingecko: string
    helius: string
} | undefined


const solscan_url = 'https://solscan.io/tx'

export const createTransactionPumpFun = async (req: Request, res: Response) => {
    //get address info from yaml files and tokens
    let pubKey, key,stuff
    let wallet: Keypair
    const walletDoc = yaml.load(fs.readFileSync('../config/cryptokeys.yaml', 'utf8'))
    wallet = Keypair.fromSecretKey(bs58.decode(process.env.PRIV_KEY || ''))
    stuff = wallet.publicKey.toBase58()
    let walletObj = walletDoc as KeyObj
    if (walletObj != undefined) {
        pubKey = walletObj.personal_wallet
    }
    const tokenDoc = yaml.load(fs.readFileSync('../config/keys.yaml', 'utf8'))
    let tokenObj = tokenDoc as TokenObj
    if (tokenObj != undefined) {
        key = tokenObj.keys?.helius
    }

    const connection = new Connection('https://mainnet.helius-rpc.com/?api-key=' + key ,'confirmed');
    let tx_info = req.body
    let message, signature, amount_in_usd, slippage
    slippage = tx_info.slippage * 100
    amount_in_usd = tx_info.solAmount * tx_info.currentPrice
    if (tx_info.type == "BUY") {
        const response = await fetch(`https://pumpportal.fun/api/trade-local`, {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify({
                "publicKey": new PublicKey(stuff),  // Your wallet public key
                "action": "buy",                 // "buy" or "sell"
                "mint": tx_info.mintName,         // contract address of the token you want to trade
                "denominatedInSol": "true",     // "true" if amount is amount of SOL, "false" if amount is number of tokens
                "amount": tx_info.solAmount,                  // amount of SOL or tokens
                "slippage": slippage,                  // percent slippage allowed
                "priorityFee": 0.001,          // priority fee 0.005 is around a 1$
                "pool": "pump"                   // exchange to trade on. "pump", "raydium" or "auto"
            })
        });
        if(response.status === 200){ // successfully generated transaction
            try {
                const data = await response.arrayBuffer();
                const tx = VersionedTransaction.deserialize(new Uint8Array(data));
                const signerKeyPair = Keypair.fromSecretKey(bs58.decode((process.env.PRIV_KEY || '')));
                tx.sign([signerKeyPair]);
                signature = await connection.sendTransaction(tx)
                message = `Wallet ${pubKey}: ${tx_info.type} operation of ${tx_info.mintName}. Sol spent:${tx_info.solAmount} (${amount_in_usd}$).`
            } catch (e) {
                res.status(500).json({
                    message: `An error happened while sending the transaction: ${e}`,
                })
                return
            }
        } else {
            console.log(response)
            res.status(500).json({
                message: `An error happened while sending the transaction to pump portal: ${response.statusText}`,
            })
            return
        }
    } else {
        const response = await fetch(`https://pumpportal.fun/api/trade-local`, {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify({
                "publicKey": new PublicKey(stuff),  // Your wallet public key
                "action": "sell",                 // "buy" or "sell"
                "mint": tx_info.mintName,         // contract address of the token you want to trade
                "denominatedInSol": "false",     // "true" if amount is amount of SOL, "false" if amount is number of tokens
                "amount": tx_info.mintAmount,                  // amount of SOL or tokens
                "slippage": slippage,                  // percent slippage allowed
                "priorityFee": 0.001,          // priority fee 0.005 is around a 1$
                "pool": "pump"                   // exchange to trade on. "pump", "raydium" or "auto"
            })
        });
        if(response.status === 200){ // successfully generated transaction
            try {
                const data = await response.arrayBuffer();
                const tx = VersionedTransaction.deserialize(new Uint8Array(data));
                const signerKeyPair = Keypair.fromSecretKey(bs58.decode((process.env.PRIV_KEY || '')));
                tx.sign([signerKeyPair]);
                signature = await connection.sendTransaction(tx)
                message = `Wallet ${pubKey}: ${tx_info.type} operation of ${tx_info.mintName}. Sol spent:${tx_info.solAmount} (${amount_in_usd}$).`
            } catch (e) {
                res.status(500).json({
                    message: `An error happened while sending the transaction: ${e}`,
                })
                return
            }
        } else {
            res.status(500).json({
                message: `An error happened while sending the transaction to pump portal: ${response.statusText}`,
            })
            return
        }
    }
    res.status(200).json({
        message: `${message}` + ` ${solscan_url}/${signature}`,
    })
    return
}
