const { rm, readFile, writeFile } = require('fs/promises');
const { resolve } = require('path');
const { promisify } = require('util');
const glob = require('glob');
const { statSync, mkdirSync } = require('fs');
const pbjs = require('protobufjs-cli/pbjs');
const pbts = require('protobufjs-cli/pbts');

const outputDir = resolve(__dirname, './src/protos');
const jsOutputFile = resolve(outputDir, 'json-module.js');
const tempFile = resolve(outputDir, 'temp.js');
const protoBaseDir = resolve(__dirname, '../proto');

const ksProtoPath = resolve(protoBaseDir, 'kitchen_sink/kitchen_sink.proto');

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

async function compileProtos(dtsOutputFile, ...args) {
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
    ksProtoPath,
  ];

  console.log(`Creating protobuf JS definitions from ${ksProtoPath}`);
  await promisify(pbjs.main)([...pbjsArgs, '--target', 'json-module', '--out', jsOutputFile]);

  console.log(`Creating protobuf TS definitions from ${ksProtoPath}`);
  try {
    await promisify(pbjs.main)([...pbjsArgs, '--target', 'static-module', '--out', tempFile]);

    // pbts internally calls jsdoc, which do strict validation of jsdoc tags.
    // Unfortunately, some protobuf comment about cron syntax contains the
    // "@every" shorthand at the begining of a line, making it appear as a
    // (invalid) jsdoc tag. Similarly, docusaurus trips on <interval> and other
    // things that looks like html tags. We fix both cases by rewriting these
    // using markdown "inline code" syntax.
    let tempFileContent = await readFile(tempFile, 'utf8');
    tempFileContent = tempFileContent.replace(
      /(@(?:yearly|monthly|weekly|daily|hourly|every))/g,
      '`$1`'
    );
    tempFileContent = tempFileContent.replace(
      /<((?:interval|phase|timezone)(?: [^>]+)?)>/g,
      '`<$1>`'
    );
    await writeFile(tempFile, tempFileContent, 'utf-8');

    await promisify(pbts.main)(['--out', dtsOutputFile, tempFile]);
  } finally {
    await rm(tempFile);
  }
}

async function main() {
  mkdirSync(outputDir, { recursive: true });

  const protoFiles = glob.sync(resolve(protoBaseDir, '**/*.proto'));
  const protosMTime = Math.max(...protoFiles.map(mtime));
  const genMTime = mtime(jsOutputFile);

  if (protosMTime < genMTime) {
    console.log('Assuming protos are up to date');
    return;
  }

  await compileProtos(
    resolve(outputDir, 'root.d.ts'),
    '--path',
    resolve(protoBaseDir, 'kitchen_sink'),
    '--path',
    resolve(protoBaseDir, 'api_upstream')
  );

  console.log('Done');
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
