import {
    Connection,
    PublicKey,
    Signer,
    Transaction,
    VersionedTransaction,
    Keypair,
    LAMPORTS_PER_SOL,
    sendAndConfirmTransaction
} from '@solana/web3.js'
import yaml from 'js-yaml'
import { Request, Response } from 'express'
import fs from 'fs'
import NodeWallet from "@coral-xyz/anchor/dist/cjs/nodewallet";
import { bs58 } from '@project-serum/anchor/dist/cjs/utils/bytes'
import dotenv from 'dotenv'
import { AnchorProvider } from "@coral-xyz/anchor";
import { PumpFunSDK } from 'pumpdotfun-sdk'



dotenv.config()


type TxRes = {
    txId: string
    solFromSell: number
} | undefined

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
    let pubKey, key
    let wallet: Keypair
    const walletDoc = yaml.load(fs.readFileSync('../config/cryptokeys.yaml', 'utf8'))
    wallet = Keypair.fromSecretKey(bs58.decode(process.env.PRIV_KEY || ''))
    let walletObj = walletDoc as KeyObj
    if (walletObj != undefined) {
        pubKey = walletObj.personal_wallet
    }
    const tokenDoc = yaml.load(fs.readFileSync('../config/keys.yaml', 'utf8'))
    let tokenObj = tokenDoc as TokenObj
    if (tokenObj != undefined) {
        key = tokenObj.keys?.helius
    }
    let wallet_conn = new NodeWallet(wallet);
    const connection = new Connection('https://mainnet.helius-rpc.com/?api-key=' + { key });
    // const connection = new Connection('https://api.mainnet-beta.solana.com')
    const provider = new AnchorProvider(connection, wallet_conn, {
        commitment: "processed",
    });

    console.log(provider)

    let sdk = new PumpFunSDK(provider);
    //create connection to the sol net

    if (!req.body) {
        res.status(400).json({
            message: 'Wrong body'
        })
        return
    }
    console.log("CHECK SDK")
    console.log(sdk)
    let tx_info = req.body
    console.log("Body received:")
    console.log(tx_info)
    console.log("BALANCE:")
    console.log("/////////////////////////////////////////////////////////////")



    let mint_amount, mintName, slippage, message, sol_amount, current_price, amount_in_usd, signature

    // let boundingCurveAccount = await sdk.getBondingCurveAccount(new PublicKey(tx_info.mintName));
    // console.log("BONDING CURVE CHECKS")
    // console.log(boundingCurveAccount)

    if (tx_info.type == "BUY") {
        mintName = new PublicKey(tx_info.mintName)
        mint_amount = tx_info.mintAmount
        slippage = BigInt(tx_info.slippage * 1000) //*1000 because it is 0.x defined on server.go
        sol_amount = tx_info.solAmount //* 1000000000 // we are leaving a bit of WSOL just to keep our account
        current_price = tx_info.currentPrice
        amount_in_usd = Math.round(tx_info.solAmount * current_price * 100) / 100
        try {
            let buyTokens = await sdk.buy(
                wallet,
                mintName,
                BigInt(LAMPORTS_PER_SOL * tx_info.solAmount),
                slippage,
                {
                    unitLimit: 200000,
                    unitPrice: 2500,
                },
                // "processed",
                // "confirmed",
            );
            console.log(buyTokens)
            if (buyTokens.success) {
                signature = buyTokens.signature
                message = `Wallet ${pubKey}: ${tx_info.type} operation of ${tx_info.mintName}. Sol spent:${tx_info.solAmount} (${amount_in_usd}$).`
            } else {
                res.status(500).json({
                    message: `An error happened while sending the transaction: ${buyTokens.error}`,
                })
            }
        } catch (e) {
            res.status(500).json({
                message: `An error happened while sending the transaction: ${e}`,
            })
            return
        }
    } else {
        mintName = new PublicKey(tx_info.mintName)
        slippage = BigInt(tx_info.slippage * 1000) //*1000 because it is 0.x defined on server.go
        mint_amount = tx_info.mintAmount
        current_price = tx_info.currentPrice
        try {
            let sellTokens = await sdk.sell(
                wallet,
                mintName,
                BigInt(tx_info.mintAmount * Math.pow(10, 6)),
                slippage,
                {
                    unitLimit: 200000,
                    unitPrice: 15000,
                },
                "finalized",
                "confirmed",
            );
            console.log(sellTokens)
            if (sellTokens.success) {
                signature = sellTokens.signature
                message = `Wallet ${pubKey}: ${tx_info.type} operation of ${tx_info.mintName}. Amount sold:${tx_info.mintAmount}.`
            } else {
                res.status(500).json({
                    message: `An error happened while sending the transaction: ${sellTokens.error}`,
                })
            }
        } catch (e) {
            res.status(500).json({
                message: `An error happened while sending the transaction: ${e}`,
            })
            return
        }

    }
    res.status(200).json({
        message: `${message}` + ` ${solscan_url}/${signature}`,
    })
    return
}
