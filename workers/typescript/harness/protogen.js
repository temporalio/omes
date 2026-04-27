// This script creates two different things from workers/proto/harness/api/api.proto:
// 1. TypeScript declarations under src/generated, used at compile time.
// 2. A copy of the raw api.proto file under each compiled output directory, used at runtime.
//
// The runtime copy is needed because src/grpc-helpers.ts registers the ProjectService with
// @grpc/grpc-js. grpc-js needs a runtime service description, not just TypeScript types.
// grpc-helpers.ts builds that description by loading ../proto/api.proto relative to the
// compiled file location. After compilation, grpc-helpers.js lives in dist/src or
// dist-test/src, so ../proto/api.proto must exist as dist/proto/api.proto or
// dist-test/proto/api.proto.


const { execFileSync } = require('node:child_process');
const { copyFileSync, mkdirSync, rmSync } = require('node:fs');
const { resolve } = require('node:path');

const packageRoot = __dirname;
const sourceProtoDir = resolve(packageRoot, '../../proto/harness/api');
const sourceProtoPath = resolve(sourceProtoDir, 'api.proto');
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

for (const runtimeOutputDir of ['dist', 'dist-test']) {
  const targetProtoDir = resolve(packageRoot, runtimeOutputDir, 'proto');
  const targetProtoPath = resolve(targetProtoDir, 'api.proto');

  mkdirSync(targetProtoDir, { recursive: true });
  copyFileSync(sourceProtoPath, targetProtoPath);
}
