const { execFileSync } = require('node:child_process');
const { mkdirSync, rmSync } = require('node:fs');
const { resolve } = require('node:path');

const packageRoot = __dirname;
const sourceProtoDir = resolve(packageRoot, '../../proto/harness/api');
const outputDir = resolve(packageRoot, 'src/generated');
const generator = require.resolve('@grpc/proto-loader/build/bin/proto-loader-gen-types.js');

rmSync(outputDir, { force: true, recursive: true });
mkdirSync(outputDir, { recursive: true });

execFileSync(
  process.execPath,
  [
    generator,
    '--grpcLib=@grpc/grpc-js',
    '--longs=Number',
    `--outDir=${outputDir}`,
    `--includeDirs=${sourceProtoDir}`,
    '--',
    'api.proto',
  ],
  {
    stdio: 'inherit',
  },
);
