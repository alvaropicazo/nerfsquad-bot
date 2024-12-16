import { createAssociatedTokenAccountInstruction, getAssociatedTokenAddress } from '@solana/spl-token';
import { Connection, Keypair, PublicKey, Transaction, VersionedTransaction, sendAndConfirmTransaction } from '@solana/web3.js';

import { Request, Response } from 'express'
import { bs58 } from '@project-serum/anchor/dist/cjs/utils/bytes'

export const createTokenAccount = async (req: Request, res: Response) => {
    if (!req.body) {
        res.status(400).json({
            message: 'Wrong body'
        })
        return
    }

    const connection = new Connection('https://api.mainnet-beta.solana.com', 'confirmed');
    const walletPublicKey = new PublicKey(req.body.address);
    const mintPublicKey = new PublicKey('So11111111111111111111111111111111111111112');

    const ataAddress = await getAssociatedTokenAddress(
        mintPublicKey,       // Mint address of the SPL token
        walletPublicKey      // Wallet address that will own the ATA
    );
    const payer = walletPublicKey; // The account paying for the creation of the ATA

    const createATAInstruction = createAssociatedTokenAccountInstruction(
        payer,              // Payer of the account creation fee
        ataAddress,         // The derived Associated Token Account address
        walletPublicKey,    // The owner of the token account
        mintPublicKey       // The token mint
    );

    console.log(createATAInstruction)

    const transaction = new Transaction().add(createATAInstruction);
    transaction.feePayer = payer; // Set the fee payer

    const { lastValidBlockHeight, blockhash } = await connection.getLatestBlockhash({
        commitment: 'finalized',
      })
    transaction.recentBlockhash = blockhash;

    // If you have a signer (e.g., Keypair for a wallet)
    const signer = Keypair.fromSecretKey(bs58.decode(process.env.PRIV_KEY || ''))

    transaction.sign(signer)
    const txId = await sendAndConfirmTransaction(connection, transaction, [signer], { skipPreflight: true })

    console.log(`transaction sending..., txId: ${txId}`)

    // const recentBlockhash =  await connection.getLatestBlockhash({
    //     commitment: 'finalized',
    // })
    // transaction.recentBlockhash = recentBlockhash.blockhash;

    // // Sign and send the transaction
    // let message = await connection.sendTransaction(transaction, [signer]);

    res.status(200).json({
        message: `Account created: ${txId}.`,
    })
    return
}