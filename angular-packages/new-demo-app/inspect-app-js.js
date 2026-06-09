import fs from 'node:fs';

async function run() {
  try {
    const res = await fetch('http://localhost:4201/app/app.ts');
    const text = await res.text();
    fs.writeFileSync('inspected-app.js', text, 'utf8');
    console.log('Saved compiled app.ts to inspected-app.js');
  } catch (err) {
    console.error('Failed to fetch:', err);
  }
}

run();
