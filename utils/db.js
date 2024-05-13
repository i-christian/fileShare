const { MongoClient } = require('mongodb');

class DBClient {
  constructor() {
    const host = process.env.DB_HOST || 'localhost';
    const port = process.env.DB_PORT || 27017;
    const database = process.env.DB_DATABASE || 'files_manager';
    const url = `mongodb://${host}:${port}/${database}`;

    this.client = new MongoClient(url, { useUnifiedTopology: true });

    // Connect to MongoDB
    this.client.connect((err) => {
      if (err) {
        console.error('MongoDB connection error:', err);
      } else {
        console.log('Connected to MongoDB');
      }
    });
  }

  // Check if the connection to MongoDB is successful
  isAlive() {
    return this.client.isConnected();
  }

  // Get the number of documents in the users collection
  async nbUsers() {
    const db = this.client.db();
    const usersCollection = db.collection('users');
    return usersCollection.countDocuments();
  }

  async createUser(email, password) {
    const db = this.client.db();
    const usersCollection = db.collection('users');
    const newUser = { email, password };
    const result = await usersCollection.insertOne(newUser);
    return result.ops[0];
  }

  async getUserByEmail(email) {
    const db = this.client.db();
    const usersCollection = db.collection('users');
    return usersCollection.findOne({ email });
  }

  // Inside the DBClient class definition
  async getUserById(userId) {
    const db = this.client.db();
    const usersCollection = db.collection('users');
    return usersCollection.findOne({ _id: userId });
  }

  // Get the number of documents in the files collection
  async nbFiles() {
    const db = this.client.db();
    const filesCollection = db.collection('files');
    return filesCollection.countDocuments();
  }
}

// Create and export an instance of DBClient called dbClient
const dbClient = new DBClient();
module.exports = dbClient;
