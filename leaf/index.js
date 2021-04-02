// Probably the worst JavaScript you'll read today.
// Unless you come across some of my Rails frontend code from 2005
const express = require('express');
const bp = require('body-parser');
const fetch = require('node-fetch');

const app = express();
const port = 6008;

app.use(bp.json());

app.post('/', (req, res) => {
  const cb = req.body.callback;
  console.log('got a request', cb);
  
  res.status(202);
  res.send('accepted');

  setTimeout(() => {
    fetch(cb, {
        method: 'POST',
    }).then(() => console.log('posted a callback'));
  }, 10000);
});

app.listen(port, () => {
  console.log(`Example app listening at http://localhost:${port}`);
});