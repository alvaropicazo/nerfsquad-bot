import { Router } from 'express'
import { createTransaction } from './v1/CreateTransaction'
import { createTokenAccount } from './v1/CreateTokenAccount'

const router = Router()

router.post('/transaction',createTransaction)
router.post('/tokenAccount',createTokenAccount)


export { router }