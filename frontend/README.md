# CyberTrap Frontend Setup Guide

This guide will help you set up the frontend for the project, built using Next.js, ShadCN UI, and TypeScript. Please follow the steps below to get started:

## 1. Set up `.env.local` file

First, create a `.env.local` file in the root of the frontend folder. This file will contain environment variables required by the application.
Example:
```bash
NEXT_PUBLIC_API_URL=http://localhost:5000/api
```

## 2. Ensure that the Backend is Set Up
Make sure that the backend is properly set up and running. The frontend relies on the backend to function correctly. If you haven't already set up the backend, please refer to the backend folder's documentation and follow the setup instructions.

## 3.  Install Dependencies
Run the following command to install the necessary dependencies for the frontend:
```bash
pnpm install
```
This will install all the required packages and dependencies.

## 4. Start the Development Server
Once the dependencies are installed, you can start the development server with the following command:
```bash
pnpm run dev
```
This will launch the Next.js development server.

## 5. Access the Application
After starting the development server, you should be able to view the application in your browser at `http://localhost:3000`. You will be required to create an account to access the portal if you haven't already done so.

## 6. Stopping the Development Server

Simply press `Ctrl + C` in the terminal where the server is running.



