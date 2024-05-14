const uuid = require('uuid');
const fs = require('fs');
const path = require('path');
const dbClient = require('../utils/db');
const redisClient = require('../utils/redis');

const FOLDER_PATH = process.env.FOLDER_PATH || '/tmp/files_manager';

const createDirectoryIfNeeded = (folderPath) => {
  if (!fs.existsSync(folderPath)) {
    fs.mkdirSync(folderPath, { recursive: true });
  }
};

const FilesController = {
  async postUpload(req, res) {
    const token = req.header('X-Token');
    if (!token) {
      return res.status(401).json({ error: 'Unauthorized' });
    }

    const userId = await redisClient.get(`auth_${token}`);
    if (!userId) {
      return res.status(401).json({ error: 'Unauthorized' });
    }

    const {
      name, type, parentId = 0, isPublic = false, data,
    } = req.body;

    if (!name) {
      return res.status(400).json({ error: 'Missing name' });
    }

    if (!type || !['folder', 'file', 'image'].includes(type)) {
      return res.status(400).json({ error: 'Missing or invalid type' });
    }

    if (type !== 'folder' && !data) {
      return res.status(400).json({ error: 'Missing data' });
    }

    if (parentId !== 0) {
      const parentFile = await dbClient.getFileById(parentId);
      if (!parentFile) {
        return res.status(400).json({ error: 'Parent not found' });
      }
      if (parentFile.type !== 'folder') {
        return res.status(400).json({ error: 'Parent is not a folder' });
      }
    }

    const file = {
      userId,
      name,
      type,
      parentId,
      isPublic,
    };

    if (type !== 'folder') {
      // Check if the data is a base64 encoded string
      if (!data.startsWith('data:image')) {
        return res.status(400).json({ error: 'Invalid image data' });
      }

      // Extract the base64 encoded data
      const imageData = data.replace(/^data:image\/\w+;base64,/, '');
      const fileData = Buffer.from(imageData, 'base64');
      const fileId = uuid.v4();
      const localPath = path.join(FOLDER_PATH, fileId);

      createDirectoryIfNeeded(FOLDER_PATH);

      try {
        // Write the file data to the local path
        fs.writeFileSync(localPath, fileData);
        file.localPath = localPath;
      } catch (err) {
        console.error('Error saving file:', err);
        return res.status(500).json({ error: 'Internal server error' });
      }
    }

    try {
      // Attempt to create a file in the database
      const newFile = await dbClient.createFile(file);
      return res.status(201).json(newFile);
    } catch (err) {
      console.error('Error creating file:', err);
      return res.status(500).json({ error: 'Internal server error' });
    }
  },
};

module.exports = FilesController;
