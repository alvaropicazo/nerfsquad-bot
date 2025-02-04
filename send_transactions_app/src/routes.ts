import { Router } from 'express'
import { createTransaction } from './v1/CreateTransaction'
import { createTokenAccount } from './v1/CreateTokenAccount'
import { createTransactionPumpFun } from './v1/CreateTransactionPumpFun'

const router = Router()

router.post('/transaction',createTransaction)
router.post('/pumpfun',createTransactionPumpFun)
router.post('/tokenAccount',createTokenAccount)


export { router }