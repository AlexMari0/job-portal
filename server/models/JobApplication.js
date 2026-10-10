import mongoose from "mongoose";

const STATUSES = ["Pending", "Accepted", "Rejected"];

const JobApplicationSchema = new mongoose.Schema({
  userId: { type: String, ref: "User", required: true },
  companyId: {
    type: mongoose.Schema.Types.ObjectId,
    ref: "Company",
    required: true,
    index: true,
  },
  jobId: {
    type: mongoose.Schema.Types.ObjectId,
    ref: "Job",
    required: true,
    index: true,
  },
  status: { type: String, default: "Pending", enum: STATUSES },
  date: { type: Number, required: true },
});

export { STATUSES };

const JobApplication = mongoose.model("JobApplication", JobApplicationSchema);

export default JobApplication;
