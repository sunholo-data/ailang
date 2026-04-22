const fs = require('fs');

const csv = `name,age,email
Alice,30,alice@example.com
Bob,25,bob@example.com
Carol,35,carol@example.com`;

fs.writeFileSync('users.csv', csv);

const lines = fs.readFileSync('users.csv', 'utf8').trim().split('\n');
const headers = lines[0].split(',');
const rows = [];

for (let i = 1; i < lines.length; i++) {
  const vals = lines[i].split(',');
  const obj = {};
  for (let j = 0; j < headers.length; j++) {
    obj[headers[j]] = vals[j];
  }
  const age = parseInt(obj.age, 10);
  if (!Number.isInteger(age) || age <= 0) continue;
  if (!obj.email.includes('@')) continue;
  obj.age = age;
  rows.push(obj);
}

fs.writeFileSync('users.json', JSON.stringify(rows, null, 2));
console.log(`Converted ${rows.length} valid rows to users.json`);
