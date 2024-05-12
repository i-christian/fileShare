const redis = require('redis');

class RedisClient {
  constructor() {
    this.client = redis.createClient();

    // Display any error of the redis client in the console
    this.client.on('error', (err) => {
      console.error('Redis Error:', err);
    });
  }

  // Check if the connection to Redis is successful
  isAlive() {
    return this.client.connected;
  }

  // Get value from Redis for a given key
  async get(key) {
    return new Promise((resolve, reject) => {
      this.client.get(key, (err, reply) => {
        if (err) {
          reject(err);
        } else {
          resolve(reply);
        }
      });
    });
  }

  // Set value in Redis for a given key with an expiration time
  async set(key, value, durationInSeconds) {
    return new Promise((resolve, reject) => {
      this.client.set(key, value, 'EX', durationInSeconds, (err, reply) => {
        if (err) {
          reject(err);
        } else {
          resolve(reply);
        }
      });
    });
  }

  // Delete value from Redis for a given key
  async del(key) {
    return new Promise((resolve, reject) => {
      this.client.del(key, (err, reply) => {
        if (err) {
          reject(err);
        } else {
          resolve(reply);
        }
      });
    });
  }
}

const redisClient = new RedisClient();
module.exports = redisClient;
