# Files Manager

This project encapsulates the core concepts covered in the ALX Software Engineering back-end trimester, including authentication, NodeJS, MongoDB, Redis, pagination, and background processing.

## Objectives

The primary objectives of this project are:

1. **User Authentication via Token**: Implement a secure authentication mechanism using tokens to authenticate users.
2. **List All Files**: Provide an endpoint to list all files available on the platform.
3. **Upload a New File**: Allow users to upload new files to the platform.
4. **Change Permission of a File**: Enable users to modify permissions associated with specific files.
5. **View a File**: Implement functionality to view files stored on the platform.
6. **Generate Thumbnails for Images**: Automatically generate thumbnails for image files to enhance user experience.

## Project Structure

The project will be structured to facilitate modularity and maintainability. Key components will be split into separate files and organized within a 'utils' folder. The project structure will adhere to best practices in software development.

## Technologies and Tools

The project will leverage the following technologies and tools:

- **Node.js**: For server-side JavaScript execution.
- **Express**: To build the API endpoints and handle HTTP requests.
- **MongoDB**: For persistent data storage.
- **Redis**: For caching and temporary data storage.
- **Bull**: To set up and utilize background processing for asynchronous tasks.
- **Mocha**: For testing.
- **Nodemon**: To monitor changes in the file system and automatically restart the server.
- **Image thumbnail**: To generate thumbnails for image files.
- **Mime-Types**: To handle MIME types of files.

## Learning Objectives

By completing this project, participants will gain proficiency in the following areas:

1. **Creating an API with Express**: Understand the fundamentals of building RESTful APIs using Express.js.
2. **Authentication Implementation**: Learn how to implement user authentication using tokens for secure access.
3. **Data Storage with MongoDB**: Acquire knowledge on storing and retrieving data from MongoDB databases.
4. **Temporary Data Storage with Redis**: Explore the usage of Redis for caching and temporary data storage.
5. **Background Worker Setup and Usage**: Understand the setup and utilization of background workers for executing tasks asynchronously.

## Requirements

- **Editors**: Allowed editors include vi, vim, emacs, Visual Studio Code.
- **Environment**: All files will be interpreted/compiled on Ubuntu 18.04 LTS using Node.js (version 12.x.x).
- **File Format**: All files should end with a new line and use the `.js` extension.
- **README.md**: A README.md file, at the root of the project folder, is mandatory and should contain project details and instructions.
- **Code Quality**: Code should adhere to linting rules using ESLint.

## Setup Instructions

To set up the project, follow these steps:

1. Clone the repository:
   ```bash
   git clone https://github.com/i-christian/alx-files_manager.git
   ```
2. Navigate to the directory
   ```
   cd alx-files_manager
   ```
3. Install dependencies
    ```
    npm install
    ```
4. Run the development server
    ```
    npm run dev
    ```
5. Alternatively, to start the production server
    ```
    npm run start-server
    ```
