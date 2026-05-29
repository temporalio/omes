const { execFileSync } = require('node:child_process');
const { copyFileSync, mkdirSync, rmSync } = require('node:fs');
const { readFile, writeFile } = require('node:fs/promises');
const { resolve } = require('node:path');
const { promisify } = require('node:util');
const pbjs = require('protobufjs-cli/pbjs');
const pbts = require('protobufjs-cli/pbts');

const packageRoot = __dirname;
const protoBaseDir = resolve(packageRoot, '../proto');

const kitchenSinkOutputDir = resolve(packageRoot, 'workerlib/kitchensink/protos');
const kitchenSinkJsOutputFile = resolve(kitchenSinkOutputDir, 'json-module.js');
const kitchenSinkDtsOutputFile = resolve(kitchenSinkOutputDir, 'root.d.ts');
const kitchenSinkTempFile = resolve(kitchenSinkOutputDir, 'temp.js');
const kitchenSinkProtoPath = resolve(protoBaseDir, 'kitchen_sink/kitchen_sink.proto');

const harnessSourceProtoDir = resolve(protoBaseDir, 'harness/api');
const harnessSourceProtoPath = resolve(harnessSourceProtoDir, 'api.proto');
const harnessOutputDir = resolve(packageRoot, 'harness/api');
const harnessGenerator = require.resolve('@grpc/proto-loader/build/bin/proto-loader-gen-types.js');

async function generateKitchenSinkProtos() {
  mkdirSync(kitchenSinkOutputDir, { recursive: true });

  const pbjsArgs = [
    '--path',
    resolve(protoBaseDir, 'kitchen_sink'),
    '--path',
    resolve(protoBaseDir, 'api_upstream'),
    '--wrap',
    'commonjs',
    '--force-long',
    '--no-verify',
    '--alt-comment',
    '--root',
    '__temporal_kitchensink',
    resolve(require.resolve('protobufjs'), '../google/protobuf/descriptor.proto'),
    kitchenSinkProtoPath,
  ];

  await promisify(pbjs.main)([
    ...pbjsArgs,
    '--target',
    'json-module',
    '--out',
    kitchenSinkJsOutputFile,
  ]);

  try {
    await promisify(pbjs.main)([
      ...pbjsArgs,
      '--target',
      'static-module',
      '--out',
      kitchenSinkTempFile,
    ]);

    // pbts validates JSDoc tags strictly, but some proto comments use cron
    // shorthands and HTML-looking placeholders that need escaping first.
    let tempFileContent = await readFile(kitchenSinkTempFile, 'utf8');
    tempFileContent = tempFileContent.replace(
      /(@(?:yearly|monthly|weekly|daily|hourly|every))/g,
      '`$1`',
    );
    tempFileContent = tempFileContent.replace(
      /<((?:interval|phase|timezone)(?: [^>]+)?)>/g,
      '`<$1>`',
    );
    await writeFile(kitchenSinkTempFile, tempFileContent, 'utf-8');

    await promisify(pbts.main)(['--out', kitchenSinkDtsOutputFile, kitchenSinkTempFile]);
    let dtsFileContent = await readFile(kitchenSinkDtsOutputFile, 'utf8');
    dtsFileContent += `
export const temporal: typeof temporal;
export const google: typeof google;
declare const root: { temporal: typeof temporal; google: typeof google };
export default root;
`;
    await writeFile(kitchenSinkDtsOutputFile, dtsFileContent, 'utf-8');
  } finally {
    rmSync(kitchenSinkTempFile, { force: true });
  }
}

function generateHarnessApiProtos() {
  rmSync(harnessOutputDir, { force: true, recursive: true });
  mkdirSync(harnessOutputDir, { recursive: true });

  execFileSync(
    process.execPath,
    [
      harnessGenerator,
      '--grpcLib=@grpc/grpc-js',
      '--longs=Number',
      `--outDir=${harnessOutputDir}`,
      `--includeDirs=${harnessSourceProtoDir}`,
      '--',
      'api.proto',
    ],
    { stdio: 'inherit' },
  );

  copyFileSync(harnessSourceProtoPath, resolve(harnessOutputDir, 'api.proto'));
}

async function main() {
  await generateKitchenSinkProtos();
  generateHarnessApiProtos();
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
