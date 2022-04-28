'use strict';

const express = require('express');

// Constants
const PORT = 8080;
const HOST = '0.0.0.0';

const MSG = process.env.MSG || "👻👻👻👻.👻"

// App
const app = express();
app.get('/', (req, res) => {
  res.send('Hello World ' + MSG);
});

app.listen(PORT, HOST);
console.log(`Running on http://${HOST}:${PORT}`);