import express from "express";
import {
  changeApplicationStatus,
  changeVisibility,
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

companyRouter.post("/register", upload.single("image"), registerCompany);
companyRouter.post("/login", loginCompany);
companyRouter.get("/company", protectCompany, getCompanyData);
companyRouter.post("/post-job", protectCompany, postJob);
companyRouter.get("/applicants", protectCompany, getCompanyJobApplicants);
companyRouter.get("/list-jobs", protectCompany, getCompanyPostedJobs);
companyRouter.post("/change-status", protectCompany, changeApplicationStatus);
companyRouter.post("/change-visibility", protectCompany, changeVisibility);

export default companyRouter;
