import express from 'express';
import start_server from './libs/boot';
import inject_routes from './routes';
import inject_middlewares from './libs/middlewares';

const server = express();

inject_middlewares(server);
inject_routes(server);
start_server(server);

export default server;
