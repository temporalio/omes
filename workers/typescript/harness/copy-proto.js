const { copyFileSync, mkdirSync } = require('node:fs');
const { resolve } = require('node:path');

const outputDirs = process.argv.slice(2);

const packageRoot = __dirname;
const sourceProtoPath = resolve(packageRoot, '../../proto/harness/api/api.proto');

// The built harness is consumed as a package dependency, so dist/dist-test need their own
// runtime proto asset instead of depending on the repo source tree layout.
for (const outputDir of outputDirs.length > 0 ? outputDirs : ['dist', 'dist-test']) {
  const targetProtoDir = resolve(packageRoot, outputDir, 'proto');
  const targetProtoPath = resolve(targetProtoDir, 'api.proto');

  mkdirSync(targetProtoDir, { recursive: true });
  copyFileSync(sourceProtoPath, targetProtoPath);
}
