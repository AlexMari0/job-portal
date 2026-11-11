import { v2 as cloudinary } from "cloudinary";
import * as companyService from "../services/companyService.js";

const cloudinaryUpload = async (filePath) => {
  const result = await cloudinary.uploader.upload(filePath);
  return result.secure_url;
};

export const registerCompany = async (req, res, next) => {
  try {
    const result = await companyService.register(
      { ...req.body, imagePath: req.file?.path },
      { uploadImage: cloudinaryUpload },
    );
    res.json({ success: true, ...result });
  } catch (e) { next(e); }
};

export const loginCompany = async (req, res, next) => {
  try {
    const result = await companyService.login(req.body);
    res.json({ success: true, ...result });
  } catch (e) { next(e); }
};

export const getCompanyData = async (req, res) => {
  res.json({ success: true, company: req.company });
};

export const postJob = async (req, res, next) => {
  try {
    const newJob = await companyService.postJob({
      ...req.body,
      companyId: req.company._id,
    });
    res.json({ success: true, newJob });
  } catch (e) { next(e); }
};

export const getCompanyJobApplicants = async (req, res, next) => {
  try {
    const applications = await companyService.getApplicants(req.company._id);
    res.json({ success: true, applications });
  } catch (e) { next(e); }
};

export const getCompanyPostedJobs = async (req, res, next) => {
  try {
    const jobsData = await companyService.getPostedJobs(req.company._id);
    res.json({ success: true, jobsData });
  } catch (e) { next(e); }
};

export const changeApplicationStatus = async (req, res, next) => {
  try {
    await companyService.changeApplicationStatus({
      applicationId: req.body.id,
      status: req.body.status,
      companyId: req.company._id,
    });
    res.json({ success: true, message: "Status Changed" });
  } catch (e) { next(e); }
};

export const changeVisibility = async (req, res, next) => {
  try {
    const job = await companyService.changeVisibility({
      jobId: req.body.id,
      companyId: req.company._id,
    });
    res.json({ success: true, job });
  } catch (e) { next(e); }
};
