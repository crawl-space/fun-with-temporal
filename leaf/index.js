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
  }, Math.floor(Math.random() * 10000)); // wait up to 10 seconds
});

const sleep = (milliseconds) => {
  return new Promise(resolve => setTimeout(resolve, milliseconds));
}

let running = 0;
app.use(async (req, res, next) => {
  console.log('running is', running);
  if (running > 0) {
    res.status(503);
    return;
  }

  running++;
  await sleep(1000);
  await next()
  running--;
})

app.post('/diff', async (req, res) => {
  await sleep(1500 + Math.floor(Math.random() * 500));
  res.status(200);
  res.send('great');
});

app.listen(port, () => {
  console.log(`Example app listening at http://localhost:${port}`);
});
