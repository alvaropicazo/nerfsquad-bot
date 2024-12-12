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
// import { NATIVE_MINT, getAssociatedTokenAddress } from '@solana/spl-token'

interface SwapCompute {
  id: string
  success: true
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
    wallet = Keypair.fromSecretKey(bs58.decode(process.env.PRIV_KEY || '',))
    // dex_wallet = dex_doc.dex[0]
    // pubKey = doc.personal_wallet
  } catch (e) {
    console.log(e);
  }

  let tx_info = req.body
  console.log("Body received:")
  console.log(tx_info)

  let min_amount_out, tokenMintExternal, tokenMintPersonal, txId, final_amount, tokenAccountPersonal, slippage, message
  if (tx_info.type == "BUY") {
    tokenMintPersonal = new PublicKey('So11111111111111111111111111111111111111112')
    tokenMintExternal = new PublicKey(tx_info.mintName)
    tokenAccountPersonal = new PublicKey(tx_info.tokenAccountPersonal)
    min_amount_out = tx_info.mintAmount
    slippage = tx_info.slippage
    final_amount = tx_info.solAmount * 1000000000 // 0.0002 * 1000000000
    message = `${tx_info.type} operation of ${tx_info.mintName}. Amount spent:  ${tx_info.solAmount}`
    try {
      txId = await executeSwap(connection, 'swap-base-in', tokenMintPersonal as PublicKey, tokenMintExternal as PublicKey, final_amount, wallet as Signer, tokenAccountPersonal, slippage)
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
    min_amount_out = tx_info.mintAmount * 1000000
    tokenAccountPersonal = new PublicKey(tx_info.tokenAccountPersonal)
    message = `${tx_info.type} operation of ${tx_info.mintName}. Amount spent:  ${tx_info.mintAmount}`
    try {
      txId = await executeSwap(connection, 'swap-base-in', tokenMintPersonal as PublicKey, tokenMintExternal as PublicKey, min_amount_out, wallet as Signer, tokenAccountPersonal, slippage)
    } catch (e) {
      res.status(500).json({
        message: 'An error happened while sending the transaction',
        e,
      })
      return
    }

  }
  res.status(200).json({
    message: `Transaction submitted: ${message}.`,
    txId,
  })
  return
}

const executeSwap = async (connection: Connection, url: string, tokenMintPersonal: PublicKey, tokenMintExternal: PublicKey, min_amount_out: number, wallet: Signer, tokenAccountPersonal: PublicKey, slippage: number) => {
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
      }/compute/${url}?inputMint=${tokenMintPersonal.toBase58()}&outputMint=${tokenMintExternal.toBase58()}&amount=${min_amount_out}&slippageBps=${slippage * 100
      }&txVersion=${txVersion}`
    )

    console.log("Data from get endpoint")
    console.log(swapResponse)

    const { data: swapTransactions } = await axios.post<{
      id: string
      version: string
      success: boolean
      data: { transaction: string }[]
    }>(`${raydium.API_URLS.SWAP_HOST}/transaction/${url}`, {
      computeUnitPriceMicroLamports: String(data.data.default.m),
      swapResponse,
      txVersion,
      wallet: wallet.publicKey.toBase58(),
      wrapSol: true,
      inputAccount: tokenAccountPersonal.toBase58(),
      // outputAccount: isOutputSol ? undefined : outputTokenAcc?.toBase58(),
    })

    console.log("Data from post transaction")
    console.log(swapTransactions)

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
        const txId = await sendAndConfirmTransaction(connection, transaction, [wallet], { skipPreflight: true })
        console.log(`${++idx} transaction confirmed, txId: ${txId}`)
        return txId
      }
    } else {
      for (const tx of allTransactions) {
        idx++
        const transaction = tx as VersionedTransaction
        transaction.sign([wallet])
        const txId = await connection.sendTransaction(tx as VersionedTransaction, { skipPreflight: true })
        const { lastValidBlockHeight, blockhash } = await connection.getLatestBlockhash({
          commitment: 'finalized',
        })
        console.log(`${idx} transaction sending..., txId: ${txId}`)
        const res = await connection.confirmTransaction(
          {
            blockhash,
            lastValidBlockHeight,
            signature: txId,
          },
          'confirmed'
        )
        console.log(`${idx} transaction confirmed`)
        return res
      }
    }
  }
  catch (e) {
    return e
  }
};