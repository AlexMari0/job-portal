import express from 'express'
import { getJobById, getJobs } from '../controllers/jobController.js';

const jobRouter = express.Router()

// Route to get all jobs data
jobRouter.get('/', getJobs)

// Route to get a single job by ID
jobRouter.get('/:id', getJobById)

export default jobRouter;