const { patchProtobufRoot } = require('@temporalio/common/lib/protobufs');
var $protobuf = require("protobufjs/light");
$protobuf.util.Long = require('long');
const unpatchedRoot = require('./json-module');
module.exports = patchProtobufRoot(unpatchedRoot);