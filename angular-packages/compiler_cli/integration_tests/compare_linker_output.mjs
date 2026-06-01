import fs from 'node:fs';
import path from 'node:path';
import { createRequire } from 'node:module';
import { pathToFileURL } from 'node:url';

const requireFromCwd = createRequire(path.join(process.cwd(), 'package.json'));

const [inputPath, goOutputPath, expectedOutputPath] = process.argv.slice(2);
if (!inputPath || !goOutputPath) {
  console.error('usage: node compare_linker_output.mjs <input.mjs> <go_output.mjs> [ngtsc_output.mjs]');
  process.exit(2);
}

const babel = requireFromCwd('@babel/core');
const linkerBabelPath = requireFromCwd.resolve('@angular/compiler-cli/linker/babel');
const { createEs2015LinkerPlugin } = await import(pathToFileURL(linkerBabelPath));

const absoluteInputPath = path.resolve(inputPath);
const code = fs.readFileSync(absoluteInputPath, 'utf8');

// 1. Run oracle (Angular linker)
const plugin = createEs2015LinkerPlugin({
  fileSystem: {
    resolve: (filePath) => filePath,
    exists: (filePath) => fs.existsSync(filePath),
    dirname: (filePath) => path.dirname(filePath),
    relative: (from, to) => path.relative(from, to),
    readFile: (filePath) => fs.readFileSync(filePath, 'utf8'),
  },
  logger: { level: 0, debug() {}, info() {}, warn() {}, error() {} },
  options: { linkerJitMode: false },
});

const expectedAst = babel.parse(code, { filename: absoluteInputPath, parserOpts: { plugins: ['typescript'] } });
const expectedResult = babel.transformFromAstSync(expectedAst, code, {
  filename: absoluteInputPath,
  plugins: [plugin],
  ast: true,
  code: Boolean(expectedOutputPath),
});
if (expectedOutputPath) {
  fs.writeFileSync(expectedOutputPath, expectedResult.code ?? '');
}

// 2. Parse Go output
const goCode = fs.readFileSync(goOutputPath, 'utf8');
const actualAst = babel.parse(goCode, { filename: goOutputPath, parserOpts: { plugins: ['typescript'] } });

// 3. Extract defineComponent / defineDirective calls
function extractDefines(ast) {
  const defines = [];
  babel.traverse(ast, {
    CallExpression(path) {
      if (
        path.node.callee.type === 'MemberExpression' &&
        path.node.callee.property.type === 'Identifier' &&
        path.node.callee.property.name.startsWith('ɵɵdefine')
      ) {
        let name = 'unknown';
        const parent = path.parentPath.node;
        if (parent.type === 'AssignmentExpression' && parent.left.type === 'MemberExpression') {
          name = parent.left.object.name + '.' + parent.left.property.name;
        } else if (parent.type === 'VariableDeclarator') {
          name = parent.id.name;
        }
        
        const generated = babel.transformFromAstSync(
          babel.types.program([babel.types.expressionStatement(path.node)]),
          '',
          { ast: false, code: true, compact: false }
        ).code;
        
        defines.push({ name, code: generated });
      }
    }
  });
  return defines;
}

const expectedDefines = extractDefines(expectedAst);
const actualDefines = extractDefines(actualAst);

// Compare
let hasError = false;
for (let i = 0; i < expectedDefines.length; i++) {
  const exp = expectedDefines[i];
  const act = actualDefines[i];
  if (!act) {
    console.error(`Missing define for ${exp.name}`);
    hasError = true;
    continue;
  }
  if (exp.code !== act.code) {
    console.error(`Mismatch in ${exp.name}:\n\nEXPECTED:\n${exp.code}\n\nACTUAL:\n${act.code}\n`);
    hasError = true;
  }
}

if (hasError) {
  process.exit(1);
}
console.log(`Matched ${expectedDefines.length} definitions!`);
