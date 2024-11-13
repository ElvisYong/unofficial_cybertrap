const fs = require('fs');

const base64String = '';  // Replace this with your base64 string

const decoded = Buffer.from(base64String, 'base64').toString('utf-8');
fs.writeFileSync('tls.crt', decoded);
console.log('tls.crt file created successfully'); 