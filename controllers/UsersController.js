const sha1 = require('sha1');
const dbClient = require('../utils/db');

const UsersController = {
  async postNew(req, res) {
    const { email, password } = req.body;

    if (!email) {
      return res.status(400).json({ error: 'Missing email' });
    }

    if (!password) {
      return res.status(400).json({ error: 'Missing password' });
    }

    const existingUser = await dbClient.getUserByEmail(email);
    if (existingUser) {
      return res.status(400).json({ error: 'Already exist' });
    }

    const hashedPassword = sha1(password);
    const newUser = await dbClient.createUser(email, hashedPassword);
    const { _id, email: userEmail } = newUser;

    return res.status(201).json({ id: _id, email: userEmail });
  },
};

module.exports = UsersController;
