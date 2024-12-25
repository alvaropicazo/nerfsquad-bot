import {
  Connection,
  PublicKey,
  Signer,
  Transaction,
  VersionedTransaction,
  Keypair,
  sendAndConfirmTransaction
} from '@solana/web3.js'
// import { getKeypairFromEnvironment } from '@solana-developers/helpers'
import yaml from 'js-yaml'
import { Request, Response } from 'express'
import fs from 'fs'
import * as raydium from '@raydium-io/raydium-sdk-v2'
// import  { Liquidity, LIQUIDITY_POOLS } from "@raydium-io/raydium-sdk";
import axios from 'axios'
import { bs58 } from '@project-serum/anchor/dist/cjs/utils/bytes'
import { exit } from 'process'
// import { NATIVE_MINT, getAssociatedTokenAddress } from '@solana/spl-token'
import dotenv from 'dotenv'
dotenv.config()
interface SwapCompute {
  id: string
  success: boolean
  version: 'V0' | 'V1'
  openTime?: undefined
  msg: undefined
  data: {
    swapType: 'BaseIn' | 'BaseOut'
    inputMint: string
    inputAmount: string
    outputMint: string
    outputAmount: string
    otherAmountThreshold: string
    slippageBps: number
    priceImpactPct: number
    routePlan: {
      poolId: string
      inputMint: string
      outputMint: string
      feeMint: string
      feeRate: number
      feeAmount: string
    }[]
  }
}

function createTimeoutSignal(timeoutMs: number): AbortSignal {
  const controller = new AbortController();
  setTimeout(() => controller.abort(), timeoutMs);
  return controller.signal;
}

const timeoutMs = 20000; // Timeout after 20 seconds
const abortSignal = createTimeoutSignal(timeoutMs);
const solscan_url = 'https://solscan.io/tx'

export const createTransaction = async (req: Request, res: Response) => {
  //create connection to the sol net
  const connection = new Connection('https://api.mainnet-beta.solana.com', 'confirmed');
  const raydiumApi = new raydium.Api({
    cluster: "mainnet",
    timeout: 5000, // (in milliseconds)
  });

  if (!req.body) {
    res.status(400).json({
      message: 'Wrong body'
    })
    return
  }
  //get address info from yaml files
  let wallet
  try {
    // const dex_doc = yaml.load(fs.readFileSync('../config/wallets.yaml', 'utf8'))
    // const doc = yaml.load(fs.readFileSync('../config/cryptokeys.yaml', 'utf8'))
    wallet = Keypair.fromSecretKey(bs58.decode(process.env.PRIV_KEY || ''))
    // dex_wallet = dex_doc.dex[0]
    // pubKey = doc.personal_wallet
  } catch (e) {
    console.log(e);
  }

  let tx_info = req.body
  console.log("Body received:")
  console.log(tx_info)

  let mint_amount, tokenMintExternal, tokenMintPersonal, txId, tokenAccountPersonal, slippage, message, sol_amount
  if (tx_info.type == "BUY") {
    tokenMintPersonal = new PublicKey('So11111111111111111111111111111111111111112')
    tokenMintExternal = new PublicKey(tx_info.mintName)
    tokenAccountPersonal = new PublicKey(tx_info.tokenAccountPersonal)
    mint_amount = tx_info.mintAmount
    slippage = tx_info.slippage
    sol_amount = ( tx_info.solAmount - 0.00001 ) * 1000000000 // we are leaving a bit of WSOL just to keep our account
    message = `${tx_info.type} operation of ${tx_info.mintName}. Amount spent:  ${tx_info.solAmount}. `
    try {
      txId = await executeSwap(tx_info.type,connection, 'swap-base-in', tokenMintPersonal as PublicKey, tokenMintExternal as PublicKey, sol_amount, mint_amount, wallet as Signer, tokenAccountPersonal, slippage)
      if (txId instanceof Error) {
        res.status(500).json({
          message: `An error happened while sending the transaction: ${txId}`,
        })
        return
      }
    } catch (e) {
      res.status(500).json({
        message: 'An error happened while sending the transaction',
        e,
      })
      return
    }
  } else {
    tokenMintPersonal = new PublicKey(tx_info.mintName)
    tokenMintExternal = new PublicKey('So11111111111111111111111111111111111111112')
    slippage = tx_info.slippage
    mint_amount = tx_info.mintAmount * 1000000
    sol_amount = tx_info.solAmount
    tokenAccountPersonal = new PublicKey(tx_info.tokenAccountPersonal)
    message = `${tx_info.type} operation of ${tx_info.mintName}. Amount spent:  ${tx_info.mintAmount}. Sol received: Around ${sol_amount} `
    try {
      txId = await executeSwap(tx_info.type, connection, 'swap-base-in', tokenMintPersonal as PublicKey, tokenMintExternal as PublicKey, mint_amount, sol_amount, wallet as Signer, tokenAccountPersonal, slippage)
      if (txId instanceof Error) {
        res.status(500).json({
          message: `An error happened while sending the transaction: ${txId}`,
        })
        return
      }
    } catch (e) {
      res.status(500).json({
        message: 'An error happened while sending the transaction',
        e,
      })
      return
    }

  }
  res.status(200).json({
    message: `Transaction submitted: ${message}.` + `${solscan_url}/${txId}`,
  })
  return
}

const executeSwap = async (tx_type: string, connection: Connection, url: string, tokenMintPersonal: PublicKey, tokenMintExternal: PublicKey, amount_to_spend: number, min_amount_out: number, wallet: Signer, tokenAccountPersonal: PublicKey, slippage: number) => {
  const txVersion = 'V0' // or LEGACY
  const isV0Tx = txVersion === 'V0'
  try {

    const { data } = await axios.get<{
      id: string
      success: boolean
      data: { default: { vh: number; h: number; m: number } }
    }>(`${raydium.API_URLS.BASE_HOST}${raydium.API_URLS.PRIORITY_FEE}`)

    const { data: swapResponse } = await axios.get<SwapCompute>(
      `${raydium.API_URLS.SWAP_HOST
      }/compute/${url}?inputMint=${tokenMintPersonal.toBase58()}&outputMint=${tokenMintExternal.toBase58()}&amount=${amount_to_spend}&slippageBps=${slippage * 100
      }&txVersion=${txVersion}`
    )

    console.log("Data from get endpoint")
    console.log(swapResponse)

    const max_permitted = (1 - slippage) * min_amount_out
    console.log(max_permitted)

    if (swapResponse.success == false) {
      throw new Error(swapResponse.msg)
    }

    //Our own slippage check
    // if (tx_type == "BUY") {
    //   if (parseInt(swapResponse.data.outputAmount) / 1000000 < max_permitted){
    //     throw new Error('Slippage Exceeded')
    //   }
    // } else {
    //   if (parseInt(swapResponse.data.outputAmount) / 1000000000 < max_permitted){
    //     throw new Error('Slippage Exceeded')
    //   }
    // }

    const { data: swapTransactions } = await axios.post<{
      id: string
      version: string
      success: boolean
      data: { transaction: string }[]
    }>(`${raydium.API_URLS.SWAP_HOST}/transaction/${url}`, {
      computeUnitPriceMicroLamports: String(data.data.default.h),
      swapResponse,
      txVersion,
      wallet: wallet.publicKey.toBase58(),
      inputAccount: tokenAccountPersonal.toBase58(),
      // outputAccount: isOutputSol ? undefined : outputTokenAcc?.toBase58(),
    })

    const allTxBuf = swapTransactions.data.map((tx) => Buffer.from(tx.transaction, 'base64'))
    const allTransactions = allTxBuf.map((txBuf) =>
      isV0Tx ? VersionedTransaction.deserialize(txBuf) : Transaction.from(txBuf)
    )

    let idx = 0
    if (!isV0Tx) {
      for (const tx of allTransactions) {
        console.log(`${++idx} transaction sending...`)
        const transaction = tx as Transaction
        transaction.sign(wallet)
        const txId = await sendAndConfirmTransaction(connection, transaction, [wallet], { skipPreflight: true, abortSignal: abortSignal })
        console.log(`${++idx} transaction confirmed, txId: ${txId}`)
        return txId
      }
    } else {
      for (const tx of allTransactions) {
        idx++
        const transaction = tx as VersionedTransaction
        transaction.sign([wallet])
        const txId = await connection.sendTransaction(tx as VersionedTransaction, { skipPreflight: true  })
        const { lastValidBlockHeight, blockhash } = await connection.getLatestBlockhash({
          commitment: 'finalized',
        })
        console.log(`${idx} transaction sending..., txId: ${txId}`)
        return txId
        // const res = await connection.confirmTransaction(
        //   {
        //     blockhash,
        //     lastValidBlockHeight,
        //     signature: txId,
        //   },
        //   'confirmed'
        // )
        // console.log(`${idx} transaction confirmed`)
        // return res
      }
    }
  }
  catch (e) {
    return e
  }
};