const { execFileSync } = require('node:child_process');
const { statSync, mkdirSync, copyFileSync, rmSync } = require('node:fs');
const { rm, readFile, writeFile } = require('node:fs/promises');
const { resolve } = require('node:path');
const { promisify } = require('node:util');
const glob = require('glob');
const pbjs = require('protobufjs-cli/pbjs');
const pbts = require('protobufjs-cli/pbts');

const packageRoot = __dirname;
const protoBaseDir = resolve(packageRoot, '../proto');

const kitchenSinkOutputDir = resolve(packageRoot, './workerlib/kitchensink/protos');
const kitchenSinkJsOutputFile = resolve(kitchenSinkOutputDir, 'json-module.js');
const kitchenSinkDtsOutputFile = resolve(kitchenSinkOutputDir, 'root.d.ts');
const kitchenSinkTempFile = resolve(kitchenSinkOutputDir, 'temp.js');
const kitchenSinkProtoPath = resolve(protoBaseDir, 'kitchen_sink/kitchen_sink.proto');

const harnessSourceProtoDir = resolve(protoBaseDir, 'harness/api');
const harnessSourceProtoPath = resolve(harnessSourceProtoDir, 'api.proto');
const harnessOutputDir = resolve(packageRoot, 'harness/api');
const harnessApiOutputFile = resolve(harnessOutputDir, 'api.ts');
const harnessGenerator = require.resolve('@grpc/proto-loader/build/bin/proto-loader-gen-types.js');
const harnessRuntimeProtoDirs = [
  resolve(packageRoot, 'lib/harness/api'),
  resolve(packageRoot, 'dist-test/harness/api'),
];

function mtime(path) {
  try {
    return statSync(path).mtimeMs;
  } catch (err) {
    if (err.code === 'ENOENT') {
      return 0;
    }
    throw err;
  }
}

function isUpToDate(inputs, outputs) {
  const newestInput = Math.max(...inputs.map(mtime));
  const oldestOutput = Math.min(...outputs.map(mtime));
  return newestInput < oldestOutput;
}

async function compileKitchenSinkProtos(...args) {
  // Use --root to avoid conflicting with user's root
  // and to avoid this error: https://github.com/protobufjs/protobuf.js/issues/1114
  const pbjsArgs = [
    ...args,
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

  console.log(`Creating protobuf JS definitions from ${kitchenSinkProtoPath}`);
  await promisify(pbjs.main)([
    ...pbjsArgs,
    '--target',
    'json-module',
    '--out',
    kitchenSinkJsOutputFile,
  ]);

  console.log(`Creating protobuf TS definitions from ${kitchenSinkProtoPath}`);
  try {
    await promisify(pbjs.main)([
      ...pbjsArgs,
      '--target',
      'static-module',
      '--out',
      kitchenSinkTempFile,
    ]);

    // pbts internally calls jsdoc, which do strict validation of jsdoc tags.
    // Unfortunately, some protobuf comment about cron syntax contains the
    // "@every" shorthand at the begining of a line, making it appear as a
    // (invalid) jsdoc tag. Similarly, docusaurus trips on <interval> and other
    // things that looks like html tags. We fix both cases by rewriting these
    // using markdown "inline code" syntax.
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
    await rm(kitchenSinkTempFile, { force: true });
  }
}

async function generateKitchenSinkProtos() {
  mkdirSync(kitchenSinkOutputDir, { recursive: true });

  const protoFiles = glob.sync(resolve(protoBaseDir, '**/*.proto'));
  if (
    isUpToDate([__filename, ...protoFiles], [kitchenSinkJsOutputFile, kitchenSinkDtsOutputFile])
  ) {
    console.log('Assuming Kitchen Sink protos are up to date');
    return;
  }

  await compileKitchenSinkProtos(
    '--path',
    resolve(protoBaseDir, 'kitchen_sink'),
    '--path',
    resolve(protoBaseDir, 'api_upstream'),
  );

  console.log('Kitchen Sink proto generation done');
}

function copyHarnessRuntimeProto() {
  for (const targetProtoDir of harnessRuntimeProtoDirs) {
    const targetProtoPath = resolve(targetProtoDir, 'api.proto');
    mkdirSync(targetProtoDir, { recursive: true });
    copyFileSync(harnessSourceProtoPath, targetProtoPath);
  }
}

function generateHarnessApiProtos() {
  const runtimeProtoOutputs = harnessRuntimeProtoDirs.map((dir) => resolve(dir, 'api.proto'));
  if (
    isUpToDate([__filename, harnessSourceProtoPath], [harnessApiOutputFile, ...runtimeProtoOutputs])
  ) {
    console.log('Assuming harness API protos are up to date');
    return;
  }

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
    {
      stdio: 'inherit',
    },
  );

  copyHarnessRuntimeProto();
  console.log('Harness API proto generation done');
}

async function main() {
  await generateKitchenSinkProtos();
  generateHarnessApiProtos();
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
