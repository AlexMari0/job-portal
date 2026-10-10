import express from "express";
import {
  ChangeJobApplicationsStatus,
  changeVisiblity,
  getCompanyData,
  getCompanyJobApplicants,
  getCompanyPostedJobs,
  loginCompany,
  postJob,
  registerCompany,
} from "../controllers/companyController.js";
import upload from "../config/multer.js";
import { protectCompany } from "../middleware/authMiddleware.js";

const companyRouter = express.Router();

// Register a company
companyRouter.post("/register", upload.single("image"), registerCompany);

// Company login
companyRouter.post("/login", loginCompany);

// Get company data
companyRouter.get("/company", protectCompany, getCompanyData);

// Post a job
companyRouter.post("/post-job", protectCompany, postJob);

// Get applicants data of company
companyRouter.get("/applicants", protectCompany, getCompanyJobApplicants);

// Get company job list
companyRouter.get("/list-jobs", protectCompany, getCompanyPostedJobs);

// Change applications status
companyRouter.post("/change-status", protectCompany, ChangeJobApplicationsStatus);

// Change applications visiblity
companyRouter.post("/change-visiblity", protectCompany, changeVisiblity);

export default companyRouter;
