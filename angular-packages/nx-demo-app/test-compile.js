const fs = require('fs');
const { spawnSync } = require('child_process');
// We can't use go-ngc easily if it's not in bin. But wait! I can just use the vite plugin logic to run a build and dump the memory outputs!
