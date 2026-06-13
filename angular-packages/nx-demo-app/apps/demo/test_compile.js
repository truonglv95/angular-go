const { compile } = require('@angular-go/build/compile');
const ts = require('typescript');
const fs = require('fs');

const config = ts.readConfigFile('tsconfig.app.json', ts.sys.readFile).config;
const parsed = ts.parseJsonConfigFileContent(config, ts.sys, '.');
compile(parsed.fileNames, parsed.options).then(res => {
  fs.writeFileSync('output.json', JSON.stringify(res, null, 2));
});
