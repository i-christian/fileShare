const dbClient = require('../utils/db');
const redisClient = require('../utils/redis');

const AppController = {
  async getStatus(req, res) {
    const redisAlive = redisClient.isAlive();
    const dbAlive = dbClient.isAlive();
    const status = { redis: redisAlive, db: dbAlive };
    const statusCode = (redisAlive && dbAlive) ? 200 : 500;
    res.status(statusCode).json(status);
  },

  async getStats(req, res) {
    try {
      const usersCount = await dbClient.nbUsers();
      const filesCount = await dbClient.nbFiles();
      const stats = { users: usersCount, files: filesCount };
      res.status(200).json(stats);
    } catch (error) {
      res.status(500).json({ error: 'Internal Server Error' });
    }
  },
};

module.exports = AppController;
