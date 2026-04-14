# 🚀 Job Portal

A full-stack **Job Portal Application** that allows users to search and apply for jobs, while recruiters can post and manage job listings.

This project is built using modern technologies with a clean and scalable architecture.

---

## 🛠️ Tech Stack

### 🔹 Frontend

* React (Vite)
* Tailwind CSS
* React Router
* Axios
* Clerk Authentication
* React Toastify
* Quill (Rich Text Editor)

### 🔹 Backend

* Node.js
* Express.js
* MongoDB (Mongoose)
* JWT Authentication
* Cloudinary (File Upload)
* Multer

### 🔹 Tools & Services

* Clerk (Authentication & Webhooks)
* Sentry (Error Monitoring)
* Vercel (Deployment)

---

## ✨ Features

### 👤 User

* Browse job listings
* Apply for jobs
* Upload resume
* Authentication with Clerk

### 🏢 Recruiter

* Login as recruiter
* Post new jobs
* Manage job listings
* View applicants

### ⚙️ General

* Role-based access (User & Recruiter)
* RESTful API
* File upload support (resume)
* Responsive UI

---

## 📁 Project Structure

```bash
project-root/
├── client/                 # Frontend (React + Vite)
│   ├── public/
│   ├── src/
│   │   ├── assets/
│   │   ├── components/
│   │   ├── context/
│   │   ├── pages/
│   │   ├── App.jsx
│   │   └── main.jsx
│   ├── index.html
│   ├── package.json
│   └── vite.config.js
│
└── server/                 # Backend (Node.js + Express)
    ├── config/
    ├── controllers/
    ├── middleware/
    ├── models/
    ├── routes/
    ├── utils/
    ├── server.js
    └── package.json
```

---

## 📦 Installation

### 1. Clone Repository

```bash
git clone https://github.com/AlexMari0/job-portal.git
cd job-portal
```

---

## ⚙️ Setup Backend

```bash
cd server
npm install
```

### ▶️ Run Backend

```bash
npm run server
```

---

## 💻 Setup Frontend

```bash
cd client
npm install
```

### ▶️ Run Frontend

```bash
npm run dev
```

---

## 🌐 Environment Variables

### 🔹 Server (`/server/.env`)

```env
PORT=5000
MONGODB_URI=your_mongodb_uri
JWT_SECRET=your_jwt_secret

CLOUDINARY_CLOUD_NAME=your_cloud_name
CLOUDINARY_API_KEY=your_api_key
CLOUDINARY_API_SECRET=your_api_secret

CLERK_SECRET_KEY=your_clerk_secret
```

### 🔹 Client (`/client/.env`)

```env
VITE_CLERK_PUBLISHABLE_KEY=your_clerk_key
VITE_API_URL=http://localhost:5000
```

---

## 🚀 Deployment

Project ini dapat dideploy menggunakan:

* Frontend: Vercel
* Backend: Vercel / Railway / Render

---

## 📸 Screenshots

<img width="1429" height="788" alt="image" src="https://github.com/user-attachments/assets/93e749e5-1407-424e-be1e-f617dac5d1c5" />

---

## 🤝 Contributing

Contributions are welcome!
Feel free to fork this repo and submit a pull request.

---

## 📄 License

This project is licensed under the MIT License.

---

## 👤 Author

* Alex Mario
* GitHub: https://github.com/Alex-Mari0
