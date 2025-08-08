const { patchProtobufRoot } = require('@temporalio/common/lib/protobufs');
var $protobuf = require("protobufjs/light");
$protobuf.util.Long = require('long');
const unpatchedRoot = require('./json-module');
const patchedRoot = patchProtobufRoot(unpatchedRoot);

// Export named exports to match TypeScript definitions
module.exports = patchedRoot;
module.exports.temporal = patchedRoot.temporal;
module.exports.google = patchedRoot.google;