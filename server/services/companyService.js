import Company from "../models/Company.js";
import Job from "../models/Job.js";
import JobApplication from "../models/JobApplication.js";
import bcrypt from "bcrypt";
import jwt from "jsonwebtoken";
import {
  ValidationError,
  AuthError,
  NotFoundError,
  ConflictError,
} from "../utils/errors.js";

const VALID_STATUSES = ["Pending", "Accepted", "Rejected"];

function generateToken(id) {
  return jwt.sign({ id }, process.env.JWT_SECRET, { expiresIn: "30d" });
}

function toCompanyDTO(company) {
  return {
    _id: company._id,
    name: company.name,
    email: company.email,
    image: company.image,
  };
}

export async function register({ name, email, password, imagePath }, { uploadImage }) {
  if (!name || !email || !password || !imagePath) {
    throw new ValidationError("Missing Details");
  }

  const existing = await Company.findOne({ email });
  if (existing) {
    throw new ConflictError("Company already registered");
  }

  const salt = await bcrypt.genSalt(10);
  const hashedPassword = await bcrypt.hash(password, salt);

  const imageUrl = await uploadImage(imagePath);

  const company = await Company.create({
    name,
    email,
    password: hashedPassword,
    image: imageUrl,
  });

  return { company: toCompanyDTO(company), token: generateToken(company._id) };
}

export async function login({ email, password }) {
  if (!email || !password) {
    throw new ValidationError("Email and password are required");
  }

  const company = await Company.findOne({ email });
  if (!company) {
    throw new AuthError();
  }

  const isMatch = await bcrypt.compare(password, company.password);
  if (!isMatch) {
    throw new AuthError();
  }

  return { company: toCompanyDTO(company), token: generateToken(company._id) };
}

export async function postJob({ title, description, location, salary, level, category, companyId }) {
  if (!title || !description || !location || !salary || !level || !category) {
    throw new ValidationError("All job fields are required");
  }

  const job = await Job.create({
    title,
    description,
    location,
    salary,
    companyId,
    date: Date.now(),
    level,
    category,
  });

  return job;
}

export async function getApplicants(companyId) {
  const applications = await JobApplication.find({ companyId })
    .populate("userId", "name image resume")
    .populate("jobId", "title location category level salary")
    .exec();

  return applications;
}

export async function getPostedJobs(companyId) {
  const [jobs, counts] = await Promise.all([
    Job.find({ companyId }),
    JobApplication.aggregate([
      { $match: { companyId } },
      { $group: { _id: "$jobId", count: { $sum: 1 } } },
    ]),
  ]);

  const countMap = new Map(counts.map((c) => [c._id.toString(), c.count]));

  return jobs.map((job) => ({
    ...job.toObject(),
    applicants: countMap.get(job._id.toString()) || 0,
  }));
}

export async function changeApplicationStatus({ applicationId, status, companyId }) {
  if (!VALID_STATUSES.includes(status)) {
    throw new ValidationError(`Invalid status. Must be one of: ${VALID_STATUSES.join(", ")}`);
  }

  const application = await JobApplication.findOneAndUpdate(
    { _id: applicationId, companyId },
    { status },
    { new: true },
  );

  if (!application) {
    throw new NotFoundError("Application not found or not authorized");
  }

  return application;
}

export async function changeVisibility({ jobId, companyId }) {
  const job = await Job.findById(jobId);
  if (!job) {
    throw new NotFoundError("Job not found");
  }

  if (companyId.toString() !== job.companyId.toString()) {
    throw new AuthError("Not authorized to modify this job");
  }

  job.visible = !job.visible;
  await job.save();

  return job;
}
