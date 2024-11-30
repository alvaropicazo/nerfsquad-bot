import { Router } from 'express'
import { createTransaction } from './v1/CreateTransaction'

const router = Router()

router.post('/transaction',createTransaction)

export { router }