const spinnerFrames = ['⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'];
let spinnerIndex = 0;
let spinnerInterval: NodeJS.Timeout | null = null;
let activeText = '';

export function startSpinner(text = 'Building...') {
  if (spinnerInterval) {
    activeText = text;
    return;
  }
  activeText = text;
  
  if (!process.stdout.isTTY) {
    process.stdout.write(`${activeText}\n`);
    return;
  }

  process.stdout.write(`⠋ ${activeText}`);
  spinnerInterval = setInterval(() => {
    spinnerIndex = (spinnerIndex + 1) % spinnerFrames.length;
    process.stdout.write(`\r${spinnerFrames[spinnerIndex]} ${activeText}`);
  }, 100);
}

export function stopSpinner() {
  if (!spinnerInterval) {
    return;
  }
  clearInterval(spinnerInterval);
  spinnerInterval = null;
  if (process.stdout.isTTY) {
    // Clear the line
    process.stdout.write('\r\x1b[K');
  }
}
