import {
  Connection,
  PublicKey,
  Signer,
  Transaction,
  VersionedTransaction,
  Keypair,
  sendAndConfirmTransaction
} from '@solana/web3.js'
import yaml from 'js-yaml'
import { Request, Response } from 'express'
import fs from 'fs'
import * as raydium from '@raydium-io/raydium-sdk-v2'
import axios from 'axios'
import { bs58 } from '@project-serum/anchor/dist/cjs/utils/bytes'
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
} | undefined

function createTimeoutSignal(timeoutMs: number): AbortSignal {
  const controller = new AbortController();
  setTimeout(() => controller.abort(), timeoutMs);
  return controller.signal;
}

const timeoutMs = 20000; // Timeout after 20 seconds
const abortSignal = createTimeoutSignal(timeoutMs);
const solscan_url = 'https://solscan.io/tx'

export const createTransaction = async (req: Request, res: Response) => {
    //get address info from yaml files and tokens
    let wallet,pubKey,tokenInstantNodes
    try {
      const walletDoc = yaml.load(fs.readFileSync('../config/cryptokeys.yaml', 'utf8'))
      wallet = Keypair.fromSecretKey(bs58.decode(process.env.PRIV_KEY || ''))
      let walletObj = walletDoc as KeyObj
      if (walletObj != undefined) {
        pubKey = walletObj.personal_wallet
      }
      const tokenDoc = yaml.load(fs.readFileSync('../config/keys.yaml', 'utf8'))
      let tokenObj = tokenDoc as TokenObj
      if (tokenObj != undefined) {
        tokenInstantNodes = tokenObj.keys?.instantnodes
      }
    } catch (e) {
      console.log(e);
    }



  //create connection to the sol net
  const connection = new Connection('https://solana-api.instantnodes.io/token-' + {tokenInstantNodes});
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

  let tx_info = req.body
  console.log("Body received:")
  console.log(tx_info)

  let mint_amount, tokenMintExternal, tokenMintPersonal, tokenAccountPersonal, slippage, message, sol_amount, current_price, amount_in_usd
  let obj:TxRes
  if (tx_info.type == "BUY") {
    tokenMintPersonal = new PublicKey('So11111111111111111111111111111111111111112')
    tokenMintExternal = new PublicKey(tx_info.mintName)
    tokenAccountPersonal = new PublicKey(tx_info.tokenAccountPersonal)
    mint_amount = tx_info.mintAmount
    slippage = tx_info.slippage
    sol_amount = ( tx_info.solAmount - 0.00001 ) * 1000000000 // we are leaving a bit of WSOL just to keep our account
    current_price = tx_info.currentPrice
    amount_in_usd = Math.round(tx_info.solAmount * current_price * 100) / 100
    message = `Wallet ${pubKey}: ${tx_info.type} operation of ${tx_info.mintName}. Sol spent:${tx_info.solAmount} (${amount_in_usd}$).`
    try {
      obj = await executeSwap(tx_info.type,connection, 'swap-base-in', tokenMintPersonal as PublicKey, tokenMintExternal as PublicKey, sol_amount, mint_amount, wallet as Signer, tokenAccountPersonal, slippage)
      if (obj instanceof Error) {
        res.status(500).json({
          message: `An error happened while sending the transaction: ${obj}`,
        })
        return
      }
    } catch (e) {
      res.status(500).json({
        message: `An error happened while sending the transaction: ${e}`,
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
    current_price = tx_info.currentPrice
    message = `Wallet ${pubKey}: ${tx_info.type} operation of ${tx_info.mintName}. Amount sold:${tx_info.mintAmount}.`
    try {
      obj = await executeSwap(tx_info.type, connection, 'swap-base-in', tokenMintPersonal as PublicKey, tokenMintExternal as PublicKey, mint_amount, sol_amount, wallet as Signer, tokenAccountPersonal, slippage)
      if (obj instanceof Error) {
        res.status(500).json({
          message: `An error happened while sending the transaction: ${obj}`,
        })
        return
      }
      if (obj != undefined) {
        let sol_from_sell_round = Math.round(obj.solFromSell * 100) / 100
        let sol_obtained = Math.round(obj.solFromSell * current_price * 100) / 100
        message = message + ` Sol received:${sol_from_sell_round} (${sol_obtained}$).`
      }
    } catch (e) {
      res.status(500).json({
        message: `An error happened while sending the transaction: ${e}`,
      })
      return
    }

  }
  res.status(200).json({
    message: `${message}` + ` ${solscan_url}/${obj?.txId}`,
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
    let sol_from_sell
    let body_req
    if (tx_type == "BUY") {
      sol_from_sell = parseInt(swapResponse.data.outputAmount) / 1000000
      body_req = {
        computeUnitPriceMicroLamports: String(data.data.default.h),
        swapResponse,
        wrapSol:true,
        unwrapSol:true,
        txVersion,
        wallet: wallet.publicKey.toBase58(),
        //inputAccount: tokenAccountPersonal.toBase58(),
        //outputAccount: isOutputSol ? undefined : outputTokenAcc?.toBase58(),
      }
      // if (parseInt(swapResponse.data.outputAmount) / 1000000 < max_permitted){
      //   throw new Error('Slippage Exceeded')
      // }
    } else {
      sol_from_sell = parseInt(swapResponse.data.outputAmount) / 1000000000
      body_req = {
        computeUnitPriceMicroLamports: String(data.data.default.h),
        swapResponse,
        wrapSol:true,
        unwrapSol:true,
        txVersion,
        wallet: wallet.publicKey.toBase58(),
        inputAccount: tokenAccountPersonal.toBase58(),
        // outputAccount: isOutputSol ? undefined : outputTokenAcc?.toBase58(),
      }
      // if (parseInt(swapResponse.data.outputAmount) / 1000000 < max_permitted){
      //   throw new Error('Slippage Exceeded')
      // }
    }

    const { data: swapTransactions } = await axios.post<{
      id: string
      version: string
      success: boolean
      msg: string
      data: { transaction: string }[]
    }>(`${raydium.API_URLS.SWAP_HOST}/transaction/${url}`, body_req)


    if (swapTransactions.success == false) {
      throw new Error(swapTransactions.msg)
    }


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
        return {txId: txId, solFromSell: sol_from_sell}
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
        return {txId: txId, solFromSell: sol_from_sell}
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
    throw String(e)
  }
};