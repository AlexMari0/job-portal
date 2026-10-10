import mongoose from "mongoose";

const CATEGORIES = [
  "Programming",
  "Data Science",
  "Designing",
  "Networking",
  "Management",
  "Marketing",
  "Cybersecurity",
];

const LOCATIONS = [
  "Bangalore",
  "Washington",
  "Hyderabad",
  "Mumbai",
  "California",
  "Chennai",
  "New York",
];

const LEVELS = [
  "Beginner level",
  "Intermediate level",
  "Senior level",
];

const jobSchema = new mongoose.Schema({
  title: { type: String, required: true, trim: true },
  description: { type: String, required: true },
  location: { type: String, required: true, enum: LOCATIONS },
  category: { type: String, required: true, enum: CATEGORIES },
  level: { type: String, required: true, enum: LEVELS },
  salary: { type: Number, required: true, min: 0 },
  date: { type: Number, required: true },
  visible: { type: Boolean, default: true },
  companyId: {
    type: mongoose.Schema.Types.ObjectId,
    ref: "Company",
    required: true,
    index: true,
  },
});

export { CATEGORIES, LOCATIONS, LEVELS };

const Job = mongoose.model("Job", jobSchema);

export default Job;
