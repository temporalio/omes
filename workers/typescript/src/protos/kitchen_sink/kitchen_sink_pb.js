// source: kitchen_sink/kitchen_sink.proto
/**
 * @fileoverview
 * @enhanceable
 * @suppress {missingRequire} reports error on implicit type usages.
 * @suppress {messageConventions} JS Compiler reports an error if a variable or
 *     field starts with 'MSG_' and isn't a translatable message.
 * @public
 */
// GENERATED CODE -- DO NOT EDIT!
/* eslint-disable */
// @ts-nocheck

var jspb = require('google-protobuf');
var goog = jspb;
var global =
    (typeof globalThis !== 'undefined' && globalThis) ||
    (typeof window !== 'undefined' && window) ||
    (typeof global !== 'undefined' && global) ||
    (typeof self !== 'undefined' && self) ||
    (function () { return this; }).call(null) ||
    Function('return this')();

var google_protobuf_duration_pb = require('google-protobuf/google/protobuf/duration_pb.js');
goog.object.extend(proto, google_protobuf_duration_pb);
var google_protobuf_empty_pb = require('google-protobuf/google/protobuf/empty_pb.js');
goog.object.extend(proto, google_protobuf_empty_pb);
var temporal_api_common_v1_message_pb = require('../temporal/api/common/v1/message_pb.js');
goog.object.extend(proto, temporal_api_common_v1_message_pb);
var temporal_api_failure_v1_message_pb = require('../temporal/api/failure/v1/message_pb.js');
goog.object.extend(proto, temporal_api_failure_v1_message_pb);
var temporal_api_enums_v1_workflow_pb = require('../temporal/api/enums/v1/workflow_pb.js');
goog.object.extend(proto, temporal_api_enums_v1_workflow_pb);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.Action', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.Action.VariantCase', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.ActionSet', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.ActivityCancellationType', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.AwaitWorkflowState', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.AwaitableChoice', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.AwaitableChoice.ConditionCase', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.CancelWorkflowAction', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.ChildWorkflowCancellationType', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.ClientAction', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.ClientAction.VariantCase', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.ClientActionSet', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.ClientSequence', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.ContinueAsNewAction', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.DoActionsUpdate', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.DoActionsUpdate.VariantCase', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.DoQuery', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.DoQuery.VariantCase', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.DoSignal', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.VariantCase', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.DoSignal.VariantCase', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.DoUpdate', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.DoUpdate.VariantCase', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.ExecuteActivityAction', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ActivityTypeCase', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.ExecuteActivityAction.LocalityCase', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.ExecuteNexusOperation', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.HandlerInvocation', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.ParentClosePolicy', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.RemoteActivityOptions', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.ReturnErrorAction', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.ReturnResultAction', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.SendSignalAction', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.SetPatchMarkerAction', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.TestInput', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.TimerAction', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.UpsertMemoAction', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.VersioningIntent', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.WithStartClientAction', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.WithStartClientAction.VariantCase', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.WorkflowInput', null, global);
goog.exportSymbol('proto.temporal.omes.kitchen_sink.WorkflowState', null, global);
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.TestInput = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.temporal.omes.kitchen_sink.TestInput, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.TestInput.displayName = 'proto.temporal.omes.kitchen_sink.TestInput';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.ClientSequence = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.temporal.omes.kitchen_sink.ClientSequence.repeatedFields_, null);
};
goog.inherits(proto.temporal.omes.kitchen_sink.ClientSequence, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.ClientSequence.displayName = 'proto.temporal.omes.kitchen_sink.ClientSequence';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.ClientActionSet = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.temporal.omes.kitchen_sink.ClientActionSet.repeatedFields_, null);
};
goog.inherits(proto.temporal.omes.kitchen_sink.ClientActionSet, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.ClientActionSet.displayName = 'proto.temporal.omes.kitchen_sink.ClientActionSet';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.WithStartClientAction = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, proto.temporal.omes.kitchen_sink.WithStartClientAction.oneofGroups_);
};
goog.inherits(proto.temporal.omes.kitchen_sink.WithStartClientAction, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.WithStartClientAction.displayName = 'proto.temporal.omes.kitchen_sink.WithStartClientAction';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.ClientAction = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, proto.temporal.omes.kitchen_sink.ClientAction.oneofGroups_);
};
goog.inherits(proto.temporal.omes.kitchen_sink.ClientAction, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.ClientAction.displayName = 'proto.temporal.omes.kitchen_sink.ClientAction';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.DoSignal = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, proto.temporal.omes.kitchen_sink.DoSignal.oneofGroups_);
};
goog.inherits(proto.temporal.omes.kitchen_sink.DoSignal, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.DoSignal.displayName = 'proto.temporal.omes.kitchen_sink.DoSignal';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.oneofGroups_);
};
goog.inherits(proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.displayName = 'proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.DoQuery = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, proto.temporal.omes.kitchen_sink.DoQuery.oneofGroups_);
};
goog.inherits(proto.temporal.omes.kitchen_sink.DoQuery, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.DoQuery.displayName = 'proto.temporal.omes.kitchen_sink.DoQuery';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.DoUpdate = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, proto.temporal.omes.kitchen_sink.DoUpdate.oneofGroups_);
};
goog.inherits(proto.temporal.omes.kitchen_sink.DoUpdate, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.DoUpdate.displayName = 'proto.temporal.omes.kitchen_sink.DoUpdate';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.DoActionsUpdate = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, proto.temporal.omes.kitchen_sink.DoActionsUpdate.oneofGroups_);
};
goog.inherits(proto.temporal.omes.kitchen_sink.DoActionsUpdate, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.DoActionsUpdate.displayName = 'proto.temporal.omes.kitchen_sink.DoActionsUpdate';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.HandlerInvocation = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.temporal.omes.kitchen_sink.HandlerInvocation.repeatedFields_, null);
};
goog.inherits(proto.temporal.omes.kitchen_sink.HandlerInvocation, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.HandlerInvocation.displayName = 'proto.temporal.omes.kitchen_sink.HandlerInvocation';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.WorkflowState = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.temporal.omes.kitchen_sink.WorkflowState, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.WorkflowState.displayName = 'proto.temporal.omes.kitchen_sink.WorkflowState';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.WorkflowInput = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.temporal.omes.kitchen_sink.WorkflowInput.repeatedFields_, null);
};
goog.inherits(proto.temporal.omes.kitchen_sink.WorkflowInput, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.WorkflowInput.displayName = 'proto.temporal.omes.kitchen_sink.WorkflowInput';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.ActionSet = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.temporal.omes.kitchen_sink.ActionSet.repeatedFields_, null);
};
goog.inherits(proto.temporal.omes.kitchen_sink.ActionSet, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.ActionSet.displayName = 'proto.temporal.omes.kitchen_sink.ActionSet';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.Action = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, proto.temporal.omes.kitchen_sink.Action.oneofGroups_);
};
goog.inherits(proto.temporal.omes.kitchen_sink.Action, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.Action.displayName = 'proto.temporal.omes.kitchen_sink.Action';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.AwaitableChoice = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, proto.temporal.omes.kitchen_sink.AwaitableChoice.oneofGroups_);
};
goog.inherits(proto.temporal.omes.kitchen_sink.AwaitableChoice, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.AwaitableChoice.displayName = 'proto.temporal.omes.kitchen_sink.AwaitableChoice';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.TimerAction = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.temporal.omes.kitchen_sink.TimerAction, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.TimerAction.displayName = 'proto.temporal.omes.kitchen_sink.TimerAction';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, proto.temporal.omes.kitchen_sink.ExecuteActivityAction.oneofGroups_);
};
goog.inherits(proto.temporal.omes.kitchen_sink.ExecuteActivityAction, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.ExecuteActivityAction.displayName = 'proto.temporal.omes.kitchen_sink.ExecuteActivityAction';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity.repeatedFields_, null);
};
goog.inherits(proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity.displayName = 'proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity.displayName = 'proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity.displayName = 'proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.repeatedFields_, null);
};
goog.inherits(proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.displayName = 'proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.AwaitWorkflowState = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.temporal.omes.kitchen_sink.AwaitWorkflowState, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.AwaitWorkflowState.displayName = 'proto.temporal.omes.kitchen_sink.AwaitWorkflowState';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.SendSignalAction = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.temporal.omes.kitchen_sink.SendSignalAction.repeatedFields_, null);
};
goog.inherits(proto.temporal.omes.kitchen_sink.SendSignalAction, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.SendSignalAction.displayName = 'proto.temporal.omes.kitchen_sink.SendSignalAction';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.CancelWorkflowAction = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.temporal.omes.kitchen_sink.CancelWorkflowAction, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.CancelWorkflowAction.displayName = 'proto.temporal.omes.kitchen_sink.CancelWorkflowAction';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.SetPatchMarkerAction = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.temporal.omes.kitchen_sink.SetPatchMarkerAction, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.SetPatchMarkerAction.displayName = 'proto.temporal.omes.kitchen_sink.SetPatchMarkerAction';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction.displayName = 'proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.UpsertMemoAction = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.temporal.omes.kitchen_sink.UpsertMemoAction, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.UpsertMemoAction.displayName = 'proto.temporal.omes.kitchen_sink.UpsertMemoAction';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.ReturnResultAction = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.temporal.omes.kitchen_sink.ReturnResultAction, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.ReturnResultAction.displayName = 'proto.temporal.omes.kitchen_sink.ReturnResultAction';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.ReturnErrorAction = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.temporal.omes.kitchen_sink.ReturnErrorAction, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.ReturnErrorAction.displayName = 'proto.temporal.omes.kitchen_sink.ReturnErrorAction';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.temporal.omes.kitchen_sink.ContinueAsNewAction.repeatedFields_, null);
};
goog.inherits(proto.temporal.omes.kitchen_sink.ContinueAsNewAction, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.ContinueAsNewAction.displayName = 'proto.temporal.omes.kitchen_sink.ContinueAsNewAction';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.RemoteActivityOptions = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.temporal.omes.kitchen_sink.RemoteActivityOptions, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.RemoteActivityOptions.displayName = 'proto.temporal.omes.kitchen_sink.RemoteActivityOptions';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.temporal.omes.kitchen_sink.ExecuteNexusOperation = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.temporal.omes.kitchen_sink.ExecuteNexusOperation, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.temporal.omes.kitchen_sink.ExecuteNexusOperation.displayName = 'proto.temporal.omes.kitchen_sink.ExecuteNexusOperation';
}



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.TestInput.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.TestInput.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.TestInput} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.TestInput.toObject = function(includeInstance, msg) {
  var f, obj = {
workflowInput: (f = msg.getWorkflowInput()) && proto.temporal.omes.kitchen_sink.WorkflowInput.toObject(includeInstance, f),
clientSequence: (f = msg.getClientSequence()) && proto.temporal.omes.kitchen_sink.ClientSequence.toObject(includeInstance, f),
withStartAction: (f = msg.getWithStartAction()) && proto.temporal.omes.kitchen_sink.WithStartClientAction.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.TestInput}
 */
proto.temporal.omes.kitchen_sink.TestInput.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.TestInput;
  return proto.temporal.omes.kitchen_sink.TestInput.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.TestInput} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.TestInput}
 */
proto.temporal.omes.kitchen_sink.TestInput.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.temporal.omes.kitchen_sink.WorkflowInput;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.WorkflowInput.deserializeBinaryFromReader);
      msg.setWorkflowInput(value);
      break;
    case 2:
      var value = new proto.temporal.omes.kitchen_sink.ClientSequence;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.ClientSequence.deserializeBinaryFromReader);
      msg.setClientSequence(value);
      break;
    case 3:
      var value = new proto.temporal.omes.kitchen_sink.WithStartClientAction;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.WithStartClientAction.deserializeBinaryFromReader);
      msg.setWithStartAction(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.TestInput.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.TestInput.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.TestInput} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.TestInput.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getWorkflowInput();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.temporal.omes.kitchen_sink.WorkflowInput.serializeBinaryToWriter
    );
  }
  f = message.getClientSequence();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.temporal.omes.kitchen_sink.ClientSequence.serializeBinaryToWriter
    );
  }
  f = message.getWithStartAction();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      proto.temporal.omes.kitchen_sink.WithStartClientAction.serializeBinaryToWriter
    );
  }
};


/**
 * optional WorkflowInput workflow_input = 1;
 * @return {?proto.temporal.omes.kitchen_sink.WorkflowInput}
 */
proto.temporal.omes.kitchen_sink.TestInput.prototype.getWorkflowInput = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.WorkflowInput} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.WorkflowInput, 1));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.WorkflowInput|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.TestInput} returns this
*/
proto.temporal.omes.kitchen_sink.TestInput.prototype.setWorkflowInput = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.TestInput} returns this
 */
proto.temporal.omes.kitchen_sink.TestInput.prototype.clearWorkflowInput = function() {
  return this.setWorkflowInput(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.TestInput.prototype.hasWorkflowInput = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional ClientSequence client_sequence = 2;
 * @return {?proto.temporal.omes.kitchen_sink.ClientSequence}
 */
proto.temporal.omes.kitchen_sink.TestInput.prototype.getClientSequence = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.ClientSequence} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.ClientSequence, 2));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.ClientSequence|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.TestInput} returns this
*/
proto.temporal.omes.kitchen_sink.TestInput.prototype.setClientSequence = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.TestInput} returns this
 */
proto.temporal.omes.kitchen_sink.TestInput.prototype.clearClientSequence = function() {
  return this.setClientSequence(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.TestInput.prototype.hasClientSequence = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional WithStartClientAction with_start_action = 3;
 * @return {?proto.temporal.omes.kitchen_sink.WithStartClientAction}
 */
proto.temporal.omes.kitchen_sink.TestInput.prototype.getWithStartAction = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.WithStartClientAction} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.WithStartClientAction, 3));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.WithStartClientAction|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.TestInput} returns this
*/
proto.temporal.omes.kitchen_sink.TestInput.prototype.setWithStartAction = function(value) {
  return jspb.Message.setWrapperField(this, 3, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.TestInput} returns this
 */
proto.temporal.omes.kitchen_sink.TestInput.prototype.clearWithStartAction = function() {
  return this.setWithStartAction(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.TestInput.prototype.hasWithStartAction = function() {
  return jspb.Message.getField(this, 3) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.temporal.omes.kitchen_sink.ClientSequence.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.ClientSequence.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.ClientSequence.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.ClientSequence} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.ClientSequence.toObject = function(includeInstance, msg) {
  var f, obj = {
actionSetsList: jspb.Message.toObjectList(msg.getActionSetsList(),
    proto.temporal.omes.kitchen_sink.ClientActionSet.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.ClientSequence}
 */
proto.temporal.omes.kitchen_sink.ClientSequence.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.ClientSequence;
  return proto.temporal.omes.kitchen_sink.ClientSequence.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.ClientSequence} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.ClientSequence}
 */
proto.temporal.omes.kitchen_sink.ClientSequence.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.temporal.omes.kitchen_sink.ClientActionSet;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.ClientActionSet.deserializeBinaryFromReader);
      msg.addActionSets(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.ClientSequence.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.ClientSequence.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.ClientSequence} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.ClientSequence.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getActionSetsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.temporal.omes.kitchen_sink.ClientActionSet.serializeBinaryToWriter
    );
  }
};


/**
 * repeated ClientActionSet action_sets = 1;
 * @return {!Array<!proto.temporal.omes.kitchen_sink.ClientActionSet>}
 */
proto.temporal.omes.kitchen_sink.ClientSequence.prototype.getActionSetsList = function() {
  return /** @type{!Array<!proto.temporal.omes.kitchen_sink.ClientActionSet>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.temporal.omes.kitchen_sink.ClientActionSet, 1));
};


/**
 * @param {!Array<!proto.temporal.omes.kitchen_sink.ClientActionSet>} value
 * @return {!proto.temporal.omes.kitchen_sink.ClientSequence} returns this
*/
proto.temporal.omes.kitchen_sink.ClientSequence.prototype.setActionSetsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.temporal.omes.kitchen_sink.ClientActionSet=} opt_value
 * @param {number=} opt_index
 * @return {!proto.temporal.omes.kitchen_sink.ClientActionSet}
 */
proto.temporal.omes.kitchen_sink.ClientSequence.prototype.addActionSets = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.temporal.omes.kitchen_sink.ClientActionSet, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.temporal.omes.kitchen_sink.ClientSequence} returns this
 */
proto.temporal.omes.kitchen_sink.ClientSequence.prototype.clearActionSetsList = function() {
  return this.setActionSetsList([]);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.temporal.omes.kitchen_sink.ClientActionSet.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.ClientActionSet.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.ClientActionSet.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.ClientActionSet} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.ClientActionSet.toObject = function(includeInstance, msg) {
  var f, obj = {
actionsList: jspb.Message.toObjectList(msg.getActionsList(),
    proto.temporal.omes.kitchen_sink.ClientAction.toObject, includeInstance),
concurrent: jspb.Message.getBooleanFieldWithDefault(msg, 2, false),
waitAtEnd: (f = msg.getWaitAtEnd()) && google_protobuf_duration_pb.Duration.toObject(includeInstance, f),
waitForCurrentRunToFinishAtEnd: jspb.Message.getBooleanFieldWithDefault(msg, 4, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.ClientActionSet}
 */
proto.temporal.omes.kitchen_sink.ClientActionSet.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.ClientActionSet;
  return proto.temporal.omes.kitchen_sink.ClientActionSet.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.ClientActionSet} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.ClientActionSet}
 */
proto.temporal.omes.kitchen_sink.ClientActionSet.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.temporal.omes.kitchen_sink.ClientAction;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.ClientAction.deserializeBinaryFromReader);
      msg.addActions(value);
      break;
    case 2:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setConcurrent(value);
      break;
    case 3:
      var value = new google_protobuf_duration_pb.Duration;
      reader.readMessage(value,google_protobuf_duration_pb.Duration.deserializeBinaryFromReader);
      msg.setWaitAtEnd(value);
      break;
    case 4:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setWaitForCurrentRunToFinishAtEnd(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.ClientActionSet.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.ClientActionSet.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.ClientActionSet} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.ClientActionSet.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getActionsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.temporal.omes.kitchen_sink.ClientAction.serializeBinaryToWriter
    );
  }
  f = message.getConcurrent();
  if (f) {
    writer.writeBool(
      2,
      f
    );
  }
  f = message.getWaitAtEnd();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      google_protobuf_duration_pb.Duration.serializeBinaryToWriter
    );
  }
  f = message.getWaitForCurrentRunToFinishAtEnd();
  if (f) {
    writer.writeBool(
      4,
      f
    );
  }
};


/**
 * repeated ClientAction actions = 1;
 * @return {!Array<!proto.temporal.omes.kitchen_sink.ClientAction>}
 */
proto.temporal.omes.kitchen_sink.ClientActionSet.prototype.getActionsList = function() {
  return /** @type{!Array<!proto.temporal.omes.kitchen_sink.ClientAction>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.temporal.omes.kitchen_sink.ClientAction, 1));
};


/**
 * @param {!Array<!proto.temporal.omes.kitchen_sink.ClientAction>} value
 * @return {!proto.temporal.omes.kitchen_sink.ClientActionSet} returns this
*/
proto.temporal.omes.kitchen_sink.ClientActionSet.prototype.setActionsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.temporal.omes.kitchen_sink.ClientAction=} opt_value
 * @param {number=} opt_index
 * @return {!proto.temporal.omes.kitchen_sink.ClientAction}
 */
proto.temporal.omes.kitchen_sink.ClientActionSet.prototype.addActions = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.temporal.omes.kitchen_sink.ClientAction, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.temporal.omes.kitchen_sink.ClientActionSet} returns this
 */
proto.temporal.omes.kitchen_sink.ClientActionSet.prototype.clearActionsList = function() {
  return this.setActionsList([]);
};


/**
 * optional bool concurrent = 2;
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ClientActionSet.prototype.getConcurrent = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 2, false));
};


/**
 * @param {boolean} value
 * @return {!proto.temporal.omes.kitchen_sink.ClientActionSet} returns this
 */
proto.temporal.omes.kitchen_sink.ClientActionSet.prototype.setConcurrent = function(value) {
  return jspb.Message.setProto3BooleanField(this, 2, value);
};


/**
 * optional google.protobuf.Duration wait_at_end = 3;
 * @return {?proto.google.protobuf.Duration}
 */
proto.temporal.omes.kitchen_sink.ClientActionSet.prototype.getWaitAtEnd = function() {
  return /** @type{?proto.google.protobuf.Duration} */ (
    jspb.Message.getWrapperField(this, google_protobuf_duration_pb.Duration, 3));
};


/**
 * @param {?proto.google.protobuf.Duration|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ClientActionSet} returns this
*/
proto.temporal.omes.kitchen_sink.ClientActionSet.prototype.setWaitAtEnd = function(value) {
  return jspb.Message.setWrapperField(this, 3, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ClientActionSet} returns this
 */
proto.temporal.omes.kitchen_sink.ClientActionSet.prototype.clearWaitAtEnd = function() {
  return this.setWaitAtEnd(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ClientActionSet.prototype.hasWaitAtEnd = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional bool wait_for_current_run_to_finish_at_end = 4;
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ClientActionSet.prototype.getWaitForCurrentRunToFinishAtEnd = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 4, false));
};


/**
 * @param {boolean} value
 * @return {!proto.temporal.omes.kitchen_sink.ClientActionSet} returns this
 */
proto.temporal.omes.kitchen_sink.ClientActionSet.prototype.setWaitForCurrentRunToFinishAtEnd = function(value) {
  return jspb.Message.setProto3BooleanField(this, 4, value);
};



/**
 * Oneof group definitions for this message. Each group defines the field
 * numbers belonging to that group. When of these fields' value is set, all
 * other fields in the group are cleared. During deserialization, if multiple
 * fields are encountered for a group, only the last value seen will be kept.
 * @private {!Array<!Array<number>>}
 * @const
 */
proto.temporal.omes.kitchen_sink.WithStartClientAction.oneofGroups_ = [[1,2]];

/**
 * @enum {number}
 */
proto.temporal.omes.kitchen_sink.WithStartClientAction.VariantCase = {
  VARIANT_NOT_SET: 0,
  DO_SIGNAL: 1,
  DO_UPDATE: 2
};

/**
 * @return {proto.temporal.omes.kitchen_sink.WithStartClientAction.VariantCase}
 */
proto.temporal.omes.kitchen_sink.WithStartClientAction.prototype.getVariantCase = function() {
  return /** @type {proto.temporal.omes.kitchen_sink.WithStartClientAction.VariantCase} */(jspb.Message.computeOneofCase(this, proto.temporal.omes.kitchen_sink.WithStartClientAction.oneofGroups_[0]));
};



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.WithStartClientAction.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.WithStartClientAction.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.WithStartClientAction} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.WithStartClientAction.toObject = function(includeInstance, msg) {
  var f, obj = {
doSignal: (f = msg.getDoSignal()) && proto.temporal.omes.kitchen_sink.DoSignal.toObject(includeInstance, f),
doUpdate: (f = msg.getDoUpdate()) && proto.temporal.omes.kitchen_sink.DoUpdate.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.WithStartClientAction}
 */
proto.temporal.omes.kitchen_sink.WithStartClientAction.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.WithStartClientAction;
  return proto.temporal.omes.kitchen_sink.WithStartClientAction.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.WithStartClientAction} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.WithStartClientAction}
 */
proto.temporal.omes.kitchen_sink.WithStartClientAction.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.temporal.omes.kitchen_sink.DoSignal;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.DoSignal.deserializeBinaryFromReader);
      msg.setDoSignal(value);
      break;
    case 2:
      var value = new proto.temporal.omes.kitchen_sink.DoUpdate;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.DoUpdate.deserializeBinaryFromReader);
      msg.setDoUpdate(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.WithStartClientAction.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.WithStartClientAction.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.WithStartClientAction} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.WithStartClientAction.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getDoSignal();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.temporal.omes.kitchen_sink.DoSignal.serializeBinaryToWriter
    );
  }
  f = message.getDoUpdate();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.temporal.omes.kitchen_sink.DoUpdate.serializeBinaryToWriter
    );
  }
};


/**
 * optional DoSignal do_signal = 1;
 * @return {?proto.temporal.omes.kitchen_sink.DoSignal}
 */
proto.temporal.omes.kitchen_sink.WithStartClientAction.prototype.getDoSignal = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.DoSignal} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.DoSignal, 1));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.DoSignal|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.WithStartClientAction} returns this
*/
proto.temporal.omes.kitchen_sink.WithStartClientAction.prototype.setDoSignal = function(value) {
  return jspb.Message.setOneofWrapperField(this, 1, proto.temporal.omes.kitchen_sink.WithStartClientAction.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.WithStartClientAction} returns this
 */
proto.temporal.omes.kitchen_sink.WithStartClientAction.prototype.clearDoSignal = function() {
  return this.setDoSignal(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.WithStartClientAction.prototype.hasDoSignal = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional DoUpdate do_update = 2;
 * @return {?proto.temporal.omes.kitchen_sink.DoUpdate}
 */
proto.temporal.omes.kitchen_sink.WithStartClientAction.prototype.getDoUpdate = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.DoUpdate} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.DoUpdate, 2));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.DoUpdate|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.WithStartClientAction} returns this
*/
proto.temporal.omes.kitchen_sink.WithStartClientAction.prototype.setDoUpdate = function(value) {
  return jspb.Message.setOneofWrapperField(this, 2, proto.temporal.omes.kitchen_sink.WithStartClientAction.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.WithStartClientAction} returns this
 */
proto.temporal.omes.kitchen_sink.WithStartClientAction.prototype.clearDoUpdate = function() {
  return this.setDoUpdate(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.WithStartClientAction.prototype.hasDoUpdate = function() {
  return jspb.Message.getField(this, 2) != null;
};



/**
 * Oneof group definitions for this message. Each group defines the field
 * numbers belonging to that group. When of these fields' value is set, all
 * other fields in the group are cleared. During deserialization, if multiple
 * fields are encountered for a group, only the last value seen will be kept.
 * @private {!Array<!Array<number>>}
 * @const
 */
proto.temporal.omes.kitchen_sink.ClientAction.oneofGroups_ = [[1,2,3,4]];

/**
 * @enum {number}
 */
proto.temporal.omes.kitchen_sink.ClientAction.VariantCase = {
  VARIANT_NOT_SET: 0,
  DO_SIGNAL: 1,
  DO_QUERY: 2,
  DO_UPDATE: 3,
  NESTED_ACTIONS: 4
};

/**
 * @return {proto.temporal.omes.kitchen_sink.ClientAction.VariantCase}
 */
proto.temporal.omes.kitchen_sink.ClientAction.prototype.getVariantCase = function() {
  return /** @type {proto.temporal.omes.kitchen_sink.ClientAction.VariantCase} */(jspb.Message.computeOneofCase(this, proto.temporal.omes.kitchen_sink.ClientAction.oneofGroups_[0]));
};



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.ClientAction.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.ClientAction.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.ClientAction} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.ClientAction.toObject = function(includeInstance, msg) {
  var f, obj = {
doSignal: (f = msg.getDoSignal()) && proto.temporal.omes.kitchen_sink.DoSignal.toObject(includeInstance, f),
doQuery: (f = msg.getDoQuery()) && proto.temporal.omes.kitchen_sink.DoQuery.toObject(includeInstance, f),
doUpdate: (f = msg.getDoUpdate()) && proto.temporal.omes.kitchen_sink.DoUpdate.toObject(includeInstance, f),
nestedActions: (f = msg.getNestedActions()) && proto.temporal.omes.kitchen_sink.ClientActionSet.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.ClientAction}
 */
proto.temporal.omes.kitchen_sink.ClientAction.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.ClientAction;
  return proto.temporal.omes.kitchen_sink.ClientAction.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.ClientAction} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.ClientAction}
 */
proto.temporal.omes.kitchen_sink.ClientAction.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.temporal.omes.kitchen_sink.DoSignal;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.DoSignal.deserializeBinaryFromReader);
      msg.setDoSignal(value);
      break;
    case 2:
      var value = new proto.temporal.omes.kitchen_sink.DoQuery;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.DoQuery.deserializeBinaryFromReader);
      msg.setDoQuery(value);
      break;
    case 3:
      var value = new proto.temporal.omes.kitchen_sink.DoUpdate;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.DoUpdate.deserializeBinaryFromReader);
      msg.setDoUpdate(value);
      break;
    case 4:
      var value = new proto.temporal.omes.kitchen_sink.ClientActionSet;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.ClientActionSet.deserializeBinaryFromReader);
      msg.setNestedActions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.ClientAction.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.ClientAction.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.ClientAction} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.ClientAction.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getDoSignal();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.temporal.omes.kitchen_sink.DoSignal.serializeBinaryToWriter
    );
  }
  f = message.getDoQuery();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.temporal.omes.kitchen_sink.DoQuery.serializeBinaryToWriter
    );
  }
  f = message.getDoUpdate();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      proto.temporal.omes.kitchen_sink.DoUpdate.serializeBinaryToWriter
    );
  }
  f = message.getNestedActions();
  if (f != null) {
    writer.writeMessage(
      4,
      f,
      proto.temporal.omes.kitchen_sink.ClientActionSet.serializeBinaryToWriter
    );
  }
};


/**
 * optional DoSignal do_signal = 1;
 * @return {?proto.temporal.omes.kitchen_sink.DoSignal}
 */
proto.temporal.omes.kitchen_sink.ClientAction.prototype.getDoSignal = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.DoSignal} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.DoSignal, 1));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.DoSignal|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ClientAction} returns this
*/
proto.temporal.omes.kitchen_sink.ClientAction.prototype.setDoSignal = function(value) {
  return jspb.Message.setOneofWrapperField(this, 1, proto.temporal.omes.kitchen_sink.ClientAction.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ClientAction} returns this
 */
proto.temporal.omes.kitchen_sink.ClientAction.prototype.clearDoSignal = function() {
  return this.setDoSignal(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ClientAction.prototype.hasDoSignal = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional DoQuery do_query = 2;
 * @return {?proto.temporal.omes.kitchen_sink.DoQuery}
 */
proto.temporal.omes.kitchen_sink.ClientAction.prototype.getDoQuery = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.DoQuery} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.DoQuery, 2));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.DoQuery|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ClientAction} returns this
*/
proto.temporal.omes.kitchen_sink.ClientAction.prototype.setDoQuery = function(value) {
  return jspb.Message.setOneofWrapperField(this, 2, proto.temporal.omes.kitchen_sink.ClientAction.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ClientAction} returns this
 */
proto.temporal.omes.kitchen_sink.ClientAction.prototype.clearDoQuery = function() {
  return this.setDoQuery(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ClientAction.prototype.hasDoQuery = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional DoUpdate do_update = 3;
 * @return {?proto.temporal.omes.kitchen_sink.DoUpdate}
 */
proto.temporal.omes.kitchen_sink.ClientAction.prototype.getDoUpdate = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.DoUpdate} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.DoUpdate, 3));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.DoUpdate|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ClientAction} returns this
*/
proto.temporal.omes.kitchen_sink.ClientAction.prototype.setDoUpdate = function(value) {
  return jspb.Message.setOneofWrapperField(this, 3, proto.temporal.omes.kitchen_sink.ClientAction.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ClientAction} returns this
 */
proto.temporal.omes.kitchen_sink.ClientAction.prototype.clearDoUpdate = function() {
  return this.setDoUpdate(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ClientAction.prototype.hasDoUpdate = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional ClientActionSet nested_actions = 4;
 * @return {?proto.temporal.omes.kitchen_sink.ClientActionSet}
 */
proto.temporal.omes.kitchen_sink.ClientAction.prototype.getNestedActions = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.ClientActionSet} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.ClientActionSet, 4));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.ClientActionSet|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ClientAction} returns this
*/
proto.temporal.omes.kitchen_sink.ClientAction.prototype.setNestedActions = function(value) {
  return jspb.Message.setOneofWrapperField(this, 4, proto.temporal.omes.kitchen_sink.ClientAction.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ClientAction} returns this
 */
proto.temporal.omes.kitchen_sink.ClientAction.prototype.clearNestedActions = function() {
  return this.setNestedActions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ClientAction.prototype.hasNestedActions = function() {
  return jspb.Message.getField(this, 4) != null;
};



/**
 * Oneof group definitions for this message. Each group defines the field
 * numbers belonging to that group. When of these fields' value is set, all
 * other fields in the group are cleared. During deserialization, if multiple
 * fields are encountered for a group, only the last value seen will be kept.
 * @private {!Array<!Array<number>>}
 * @const
 */
proto.temporal.omes.kitchen_sink.DoSignal.oneofGroups_ = [[1,2]];

/**
 * @enum {number}
 */
proto.temporal.omes.kitchen_sink.DoSignal.VariantCase = {
  VARIANT_NOT_SET: 0,
  DO_SIGNAL_ACTIONS: 1,
  CUSTOM: 2
};

/**
 * @return {proto.temporal.omes.kitchen_sink.DoSignal.VariantCase}
 */
proto.temporal.omes.kitchen_sink.DoSignal.prototype.getVariantCase = function() {
  return /** @type {proto.temporal.omes.kitchen_sink.DoSignal.VariantCase} */(jspb.Message.computeOneofCase(this, proto.temporal.omes.kitchen_sink.DoSignal.oneofGroups_[0]));
};



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.DoSignal.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.DoSignal.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.DoSignal} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.DoSignal.toObject = function(includeInstance, msg) {
  var f, obj = {
doSignalActions: (f = msg.getDoSignalActions()) && proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.toObject(includeInstance, f),
custom: (f = msg.getCustom()) && proto.temporal.omes.kitchen_sink.HandlerInvocation.toObject(includeInstance, f),
withStart: jspb.Message.getBooleanFieldWithDefault(msg, 3, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.DoSignal}
 */
proto.temporal.omes.kitchen_sink.DoSignal.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.DoSignal;
  return proto.temporal.omes.kitchen_sink.DoSignal.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.DoSignal} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.DoSignal}
 */
proto.temporal.omes.kitchen_sink.DoSignal.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.deserializeBinaryFromReader);
      msg.setDoSignalActions(value);
      break;
    case 2:
      var value = new proto.temporal.omes.kitchen_sink.HandlerInvocation;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.HandlerInvocation.deserializeBinaryFromReader);
      msg.setCustom(value);
      break;
    case 3:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setWithStart(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.DoSignal.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.DoSignal.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.DoSignal} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.DoSignal.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getDoSignalActions();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.serializeBinaryToWriter
    );
  }
  f = message.getCustom();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.temporal.omes.kitchen_sink.HandlerInvocation.serializeBinaryToWriter
    );
  }
  f = message.getWithStart();
  if (f) {
    writer.writeBool(
      3,
      f
    );
  }
};



/**
 * Oneof group definitions for this message. Each group defines the field
 * numbers belonging to that group. When of these fields' value is set, all
 * other fields in the group are cleared. During deserialization, if multiple
 * fields are encountered for a group, only the last value seen will be kept.
 * @private {!Array<!Array<number>>}
 * @const
 */
proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.oneofGroups_ = [[1,2]];

/**
 * @enum {number}
 */
proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.VariantCase = {
  VARIANT_NOT_SET: 0,
  DO_ACTIONS: 1,
  DO_ACTIONS_IN_MAIN: 2
};

/**
 * @return {proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.VariantCase}
 */
proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.prototype.getVariantCase = function() {
  return /** @type {proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.VariantCase} */(jspb.Message.computeOneofCase(this, proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.oneofGroups_[0]));
};



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.toObject = function(includeInstance, msg) {
  var f, obj = {
doActions: (f = msg.getDoActions()) && proto.temporal.omes.kitchen_sink.ActionSet.toObject(includeInstance, f),
doActionsInMain: (f = msg.getDoActionsInMain()) && proto.temporal.omes.kitchen_sink.ActionSet.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions}
 */
proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions;
  return proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions}
 */
proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.temporal.omes.kitchen_sink.ActionSet;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.ActionSet.deserializeBinaryFromReader);
      msg.setDoActions(value);
      break;
    case 2:
      var value = new proto.temporal.omes.kitchen_sink.ActionSet;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.ActionSet.deserializeBinaryFromReader);
      msg.setDoActionsInMain(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getDoActions();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.temporal.omes.kitchen_sink.ActionSet.serializeBinaryToWriter
    );
  }
  f = message.getDoActionsInMain();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.temporal.omes.kitchen_sink.ActionSet.serializeBinaryToWriter
    );
  }
};


/**
 * optional ActionSet do_actions = 1;
 * @return {?proto.temporal.omes.kitchen_sink.ActionSet}
 */
proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.prototype.getDoActions = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.ActionSet} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.ActionSet, 1));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.ActionSet|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions} returns this
*/
proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.prototype.setDoActions = function(value) {
  return jspb.Message.setOneofWrapperField(this, 1, proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions} returns this
 */
proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.prototype.clearDoActions = function() {
  return this.setDoActions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.prototype.hasDoActions = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional ActionSet do_actions_in_main = 2;
 * @return {?proto.temporal.omes.kitchen_sink.ActionSet}
 */
proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.prototype.getDoActionsInMain = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.ActionSet} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.ActionSet, 2));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.ActionSet|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions} returns this
*/
proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.prototype.setDoActionsInMain = function(value) {
  return jspb.Message.setOneofWrapperField(this, 2, proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions} returns this
 */
proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.prototype.clearDoActionsInMain = function() {
  return this.setDoActionsInMain(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions.prototype.hasDoActionsInMain = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional DoSignalActions do_signal_actions = 1;
 * @return {?proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions}
 */
proto.temporal.omes.kitchen_sink.DoSignal.prototype.getDoSignalActions = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions, 1));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.DoSignal.DoSignalActions|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.DoSignal} returns this
*/
proto.temporal.omes.kitchen_sink.DoSignal.prototype.setDoSignalActions = function(value) {
  return jspb.Message.setOneofWrapperField(this, 1, proto.temporal.omes.kitchen_sink.DoSignal.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.DoSignal} returns this
 */
proto.temporal.omes.kitchen_sink.DoSignal.prototype.clearDoSignalActions = function() {
  return this.setDoSignalActions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.DoSignal.prototype.hasDoSignalActions = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional HandlerInvocation custom = 2;
 * @return {?proto.temporal.omes.kitchen_sink.HandlerInvocation}
 */
proto.temporal.omes.kitchen_sink.DoSignal.prototype.getCustom = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.HandlerInvocation} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.HandlerInvocation, 2));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.HandlerInvocation|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.DoSignal} returns this
*/
proto.temporal.omes.kitchen_sink.DoSignal.prototype.setCustom = function(value) {
  return jspb.Message.setOneofWrapperField(this, 2, proto.temporal.omes.kitchen_sink.DoSignal.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.DoSignal} returns this
 */
proto.temporal.omes.kitchen_sink.DoSignal.prototype.clearCustom = function() {
  return this.setCustom(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.DoSignal.prototype.hasCustom = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional bool with_start = 3;
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.DoSignal.prototype.getWithStart = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 3, false));
};


/**
 * @param {boolean} value
 * @return {!proto.temporal.omes.kitchen_sink.DoSignal} returns this
 */
proto.temporal.omes.kitchen_sink.DoSignal.prototype.setWithStart = function(value) {
  return jspb.Message.setProto3BooleanField(this, 3, value);
};



/**
 * Oneof group definitions for this message. Each group defines the field
 * numbers belonging to that group. When of these fields' value is set, all
 * other fields in the group are cleared. During deserialization, if multiple
 * fields are encountered for a group, only the last value seen will be kept.
 * @private {!Array<!Array<number>>}
 * @const
 */
proto.temporal.omes.kitchen_sink.DoQuery.oneofGroups_ = [[1,2]];

/**
 * @enum {number}
 */
proto.temporal.omes.kitchen_sink.DoQuery.VariantCase = {
  VARIANT_NOT_SET: 0,
  REPORT_STATE: 1,
  CUSTOM: 2
};

/**
 * @return {proto.temporal.omes.kitchen_sink.DoQuery.VariantCase}
 */
proto.temporal.omes.kitchen_sink.DoQuery.prototype.getVariantCase = function() {
  return /** @type {proto.temporal.omes.kitchen_sink.DoQuery.VariantCase} */(jspb.Message.computeOneofCase(this, proto.temporal.omes.kitchen_sink.DoQuery.oneofGroups_[0]));
};



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.DoQuery.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.DoQuery.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.DoQuery} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.DoQuery.toObject = function(includeInstance, msg) {
  var f, obj = {
reportState: (f = msg.getReportState()) && temporal_api_common_v1_message_pb.Payloads.toObject(includeInstance, f),
custom: (f = msg.getCustom()) && proto.temporal.omes.kitchen_sink.HandlerInvocation.toObject(includeInstance, f),
failureExpected: jspb.Message.getBooleanFieldWithDefault(msg, 10, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.DoQuery}
 */
proto.temporal.omes.kitchen_sink.DoQuery.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.DoQuery;
  return proto.temporal.omes.kitchen_sink.DoQuery.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.DoQuery} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.DoQuery}
 */
proto.temporal.omes.kitchen_sink.DoQuery.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new temporal_api_common_v1_message_pb.Payloads;
      reader.readMessage(value,temporal_api_common_v1_message_pb.Payloads.deserializeBinaryFromReader);
      msg.setReportState(value);
      break;
    case 2:
      var value = new proto.temporal.omes.kitchen_sink.HandlerInvocation;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.HandlerInvocation.deserializeBinaryFromReader);
      msg.setCustom(value);
      break;
    case 10:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setFailureExpected(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.DoQuery.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.DoQuery.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.DoQuery} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.DoQuery.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getReportState();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      temporal_api_common_v1_message_pb.Payloads.serializeBinaryToWriter
    );
  }
  f = message.getCustom();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.temporal.omes.kitchen_sink.HandlerInvocation.serializeBinaryToWriter
    );
  }
  f = message.getFailureExpected();
  if (f) {
    writer.writeBool(
      10,
      f
    );
  }
};


/**
 * optional temporal.api.common.v1.Payloads report_state = 1;
 * @return {?proto.temporal.api.common.v1.Payloads}
 */
proto.temporal.omes.kitchen_sink.DoQuery.prototype.getReportState = function() {
  return /** @type{?proto.temporal.api.common.v1.Payloads} */ (
    jspb.Message.getWrapperField(this, temporal_api_common_v1_message_pb.Payloads, 1));
};


/**
 * @param {?proto.temporal.api.common.v1.Payloads|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.DoQuery} returns this
*/
proto.temporal.omes.kitchen_sink.DoQuery.prototype.setReportState = function(value) {
  return jspb.Message.setOneofWrapperField(this, 1, proto.temporal.omes.kitchen_sink.DoQuery.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.DoQuery} returns this
 */
proto.temporal.omes.kitchen_sink.DoQuery.prototype.clearReportState = function() {
  return this.setReportState(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.DoQuery.prototype.hasReportState = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional HandlerInvocation custom = 2;
 * @return {?proto.temporal.omes.kitchen_sink.HandlerInvocation}
 */
proto.temporal.omes.kitchen_sink.DoQuery.prototype.getCustom = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.HandlerInvocation} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.HandlerInvocation, 2));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.HandlerInvocation|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.DoQuery} returns this
*/
proto.temporal.omes.kitchen_sink.DoQuery.prototype.setCustom = function(value) {
  return jspb.Message.setOneofWrapperField(this, 2, proto.temporal.omes.kitchen_sink.DoQuery.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.DoQuery} returns this
 */
proto.temporal.omes.kitchen_sink.DoQuery.prototype.clearCustom = function() {
  return this.setCustom(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.DoQuery.prototype.hasCustom = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional bool failure_expected = 10;
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.DoQuery.prototype.getFailureExpected = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 10, false));
};


/**
 * @param {boolean} value
 * @return {!proto.temporal.omes.kitchen_sink.DoQuery} returns this
 */
proto.temporal.omes.kitchen_sink.DoQuery.prototype.setFailureExpected = function(value) {
  return jspb.Message.setProto3BooleanField(this, 10, value);
};



/**
 * Oneof group definitions for this message. Each group defines the field
 * numbers belonging to that group. When of these fields' value is set, all
 * other fields in the group are cleared. During deserialization, if multiple
 * fields are encountered for a group, only the last value seen will be kept.
 * @private {!Array<!Array<number>>}
 * @const
 */
proto.temporal.omes.kitchen_sink.DoUpdate.oneofGroups_ = [[1,2]];

/**
 * @enum {number}
 */
proto.temporal.omes.kitchen_sink.DoUpdate.VariantCase = {
  VARIANT_NOT_SET: 0,
  DO_ACTIONS: 1,
  CUSTOM: 2
};

/**
 * @return {proto.temporal.omes.kitchen_sink.DoUpdate.VariantCase}
 */
proto.temporal.omes.kitchen_sink.DoUpdate.prototype.getVariantCase = function() {
  return /** @type {proto.temporal.omes.kitchen_sink.DoUpdate.VariantCase} */(jspb.Message.computeOneofCase(this, proto.temporal.omes.kitchen_sink.DoUpdate.oneofGroups_[0]));
};



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.DoUpdate.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.DoUpdate.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.DoUpdate} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.DoUpdate.toObject = function(includeInstance, msg) {
  var f, obj = {
doActions: (f = msg.getDoActions()) && proto.temporal.omes.kitchen_sink.DoActionsUpdate.toObject(includeInstance, f),
custom: (f = msg.getCustom()) && proto.temporal.omes.kitchen_sink.HandlerInvocation.toObject(includeInstance, f),
withStart: jspb.Message.getBooleanFieldWithDefault(msg, 3, false),
failureExpected: jspb.Message.getBooleanFieldWithDefault(msg, 10, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.DoUpdate}
 */
proto.temporal.omes.kitchen_sink.DoUpdate.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.DoUpdate;
  return proto.temporal.omes.kitchen_sink.DoUpdate.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.DoUpdate} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.DoUpdate}
 */
proto.temporal.omes.kitchen_sink.DoUpdate.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.temporal.omes.kitchen_sink.DoActionsUpdate;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.DoActionsUpdate.deserializeBinaryFromReader);
      msg.setDoActions(value);
      break;
    case 2:
      var value = new proto.temporal.omes.kitchen_sink.HandlerInvocation;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.HandlerInvocation.deserializeBinaryFromReader);
      msg.setCustom(value);
      break;
    case 3:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setWithStart(value);
      break;
    case 10:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setFailureExpected(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.DoUpdate.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.DoUpdate.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.DoUpdate} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.DoUpdate.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getDoActions();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.temporal.omes.kitchen_sink.DoActionsUpdate.serializeBinaryToWriter
    );
  }
  f = message.getCustom();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.temporal.omes.kitchen_sink.HandlerInvocation.serializeBinaryToWriter
    );
  }
  f = message.getWithStart();
  if (f) {
    writer.writeBool(
      3,
      f
    );
  }
  f = message.getFailureExpected();
  if (f) {
    writer.writeBool(
      10,
      f
    );
  }
};


/**
 * optional DoActionsUpdate do_actions = 1;
 * @return {?proto.temporal.omes.kitchen_sink.DoActionsUpdate}
 */
proto.temporal.omes.kitchen_sink.DoUpdate.prototype.getDoActions = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.DoActionsUpdate} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.DoActionsUpdate, 1));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.DoActionsUpdate|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.DoUpdate} returns this
*/
proto.temporal.omes.kitchen_sink.DoUpdate.prototype.setDoActions = function(value) {
  return jspb.Message.setOneofWrapperField(this, 1, proto.temporal.omes.kitchen_sink.DoUpdate.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.DoUpdate} returns this
 */
proto.temporal.omes.kitchen_sink.DoUpdate.prototype.clearDoActions = function() {
  return this.setDoActions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.DoUpdate.prototype.hasDoActions = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional HandlerInvocation custom = 2;
 * @return {?proto.temporal.omes.kitchen_sink.HandlerInvocation}
 */
proto.temporal.omes.kitchen_sink.DoUpdate.prototype.getCustom = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.HandlerInvocation} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.HandlerInvocation, 2));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.HandlerInvocation|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.DoUpdate} returns this
*/
proto.temporal.omes.kitchen_sink.DoUpdate.prototype.setCustom = function(value) {
  return jspb.Message.setOneofWrapperField(this, 2, proto.temporal.omes.kitchen_sink.DoUpdate.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.DoUpdate} returns this
 */
proto.temporal.omes.kitchen_sink.DoUpdate.prototype.clearCustom = function() {
  return this.setCustom(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.DoUpdate.prototype.hasCustom = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional bool with_start = 3;
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.DoUpdate.prototype.getWithStart = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 3, false));
};


/**
 * @param {boolean} value
 * @return {!proto.temporal.omes.kitchen_sink.DoUpdate} returns this
 */
proto.temporal.omes.kitchen_sink.DoUpdate.prototype.setWithStart = function(value) {
  return jspb.Message.setProto3BooleanField(this, 3, value);
};


/**
 * optional bool failure_expected = 10;
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.DoUpdate.prototype.getFailureExpected = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 10, false));
};


/**
 * @param {boolean} value
 * @return {!proto.temporal.omes.kitchen_sink.DoUpdate} returns this
 */
proto.temporal.omes.kitchen_sink.DoUpdate.prototype.setFailureExpected = function(value) {
  return jspb.Message.setProto3BooleanField(this, 10, value);
};



/**
 * Oneof group definitions for this message. Each group defines the field
 * numbers belonging to that group. When of these fields' value is set, all
 * other fields in the group are cleared. During deserialization, if multiple
 * fields are encountered for a group, only the last value seen will be kept.
 * @private {!Array<!Array<number>>}
 * @const
 */
proto.temporal.omes.kitchen_sink.DoActionsUpdate.oneofGroups_ = [[1,2]];

/**
 * @enum {number}
 */
proto.temporal.omes.kitchen_sink.DoActionsUpdate.VariantCase = {
  VARIANT_NOT_SET: 0,
  DO_ACTIONS: 1,
  REJECT_ME: 2
};

/**
 * @return {proto.temporal.omes.kitchen_sink.DoActionsUpdate.VariantCase}
 */
proto.temporal.omes.kitchen_sink.DoActionsUpdate.prototype.getVariantCase = function() {
  return /** @type {proto.temporal.omes.kitchen_sink.DoActionsUpdate.VariantCase} */(jspb.Message.computeOneofCase(this, proto.temporal.omes.kitchen_sink.DoActionsUpdate.oneofGroups_[0]));
};



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.DoActionsUpdate.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.DoActionsUpdate.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.DoActionsUpdate} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.DoActionsUpdate.toObject = function(includeInstance, msg) {
  var f, obj = {
doActions: (f = msg.getDoActions()) && proto.temporal.omes.kitchen_sink.ActionSet.toObject(includeInstance, f),
rejectMe: (f = msg.getRejectMe()) && google_protobuf_empty_pb.Empty.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.DoActionsUpdate}
 */
proto.temporal.omes.kitchen_sink.DoActionsUpdate.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.DoActionsUpdate;
  return proto.temporal.omes.kitchen_sink.DoActionsUpdate.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.DoActionsUpdate} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.DoActionsUpdate}
 */
proto.temporal.omes.kitchen_sink.DoActionsUpdate.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.temporal.omes.kitchen_sink.ActionSet;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.ActionSet.deserializeBinaryFromReader);
      msg.setDoActions(value);
      break;
    case 2:
      var value = new google_protobuf_empty_pb.Empty;
      reader.readMessage(value,google_protobuf_empty_pb.Empty.deserializeBinaryFromReader);
      msg.setRejectMe(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.DoActionsUpdate.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.DoActionsUpdate.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.DoActionsUpdate} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.DoActionsUpdate.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getDoActions();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.temporal.omes.kitchen_sink.ActionSet.serializeBinaryToWriter
    );
  }
  f = message.getRejectMe();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      google_protobuf_empty_pb.Empty.serializeBinaryToWriter
    );
  }
};


/**
 * optional ActionSet do_actions = 1;
 * @return {?proto.temporal.omes.kitchen_sink.ActionSet}
 */
proto.temporal.omes.kitchen_sink.DoActionsUpdate.prototype.getDoActions = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.ActionSet} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.ActionSet, 1));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.ActionSet|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.DoActionsUpdate} returns this
*/
proto.temporal.omes.kitchen_sink.DoActionsUpdate.prototype.setDoActions = function(value) {
  return jspb.Message.setOneofWrapperField(this, 1, proto.temporal.omes.kitchen_sink.DoActionsUpdate.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.DoActionsUpdate} returns this
 */
proto.temporal.omes.kitchen_sink.DoActionsUpdate.prototype.clearDoActions = function() {
  return this.setDoActions(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.DoActionsUpdate.prototype.hasDoActions = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional google.protobuf.Empty reject_me = 2;
 * @return {?proto.google.protobuf.Empty}
 */
proto.temporal.omes.kitchen_sink.DoActionsUpdate.prototype.getRejectMe = function() {
  return /** @type{?proto.google.protobuf.Empty} */ (
    jspb.Message.getWrapperField(this, google_protobuf_empty_pb.Empty, 2));
};


/**
 * @param {?proto.google.protobuf.Empty|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.DoActionsUpdate} returns this
*/
proto.temporal.omes.kitchen_sink.DoActionsUpdate.prototype.setRejectMe = function(value) {
  return jspb.Message.setOneofWrapperField(this, 2, proto.temporal.omes.kitchen_sink.DoActionsUpdate.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.DoActionsUpdate} returns this
 */
proto.temporal.omes.kitchen_sink.DoActionsUpdate.prototype.clearRejectMe = function() {
  return this.setRejectMe(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.DoActionsUpdate.prototype.hasRejectMe = function() {
  return jspb.Message.getField(this, 2) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.temporal.omes.kitchen_sink.HandlerInvocation.repeatedFields_ = [2];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.HandlerInvocation.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.HandlerInvocation.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.HandlerInvocation} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.HandlerInvocation.toObject = function(includeInstance, msg) {
  var f, obj = {
name: jspb.Message.getFieldWithDefault(msg, 1, ""),
argsList: jspb.Message.toObjectList(msg.getArgsList(),
    temporal_api_common_v1_message_pb.Payload.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.HandlerInvocation}
 */
proto.temporal.omes.kitchen_sink.HandlerInvocation.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.HandlerInvocation;
  return proto.temporal.omes.kitchen_sink.HandlerInvocation.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.HandlerInvocation} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.HandlerInvocation}
 */
proto.temporal.omes.kitchen_sink.HandlerInvocation.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 2:
      var value = new temporal_api_common_v1_message_pb.Payload;
      reader.readMessage(value,temporal_api_common_v1_message_pb.Payload.deserializeBinaryFromReader);
      msg.addArgs(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.HandlerInvocation.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.HandlerInvocation.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.HandlerInvocation} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.HandlerInvocation.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getName();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getArgsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      2,
      f,
      temporal_api_common_v1_message_pb.Payload.serializeBinaryToWriter
    );
  }
};


/**
 * optional string name = 1;
 * @return {string}
 */
proto.temporal.omes.kitchen_sink.HandlerInvocation.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.temporal.omes.kitchen_sink.HandlerInvocation} returns this
 */
proto.temporal.omes.kitchen_sink.HandlerInvocation.prototype.setName = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * repeated temporal.api.common.v1.Payload args = 2;
 * @return {!Array<!proto.temporal.api.common.v1.Payload>}
 */
proto.temporal.omes.kitchen_sink.HandlerInvocation.prototype.getArgsList = function() {
  return /** @type{!Array<!proto.temporal.api.common.v1.Payload>} */ (
    jspb.Message.getRepeatedWrapperField(this, temporal_api_common_v1_message_pb.Payload, 2));
};


/**
 * @param {!Array<!proto.temporal.api.common.v1.Payload>} value
 * @return {!proto.temporal.omes.kitchen_sink.HandlerInvocation} returns this
*/
proto.temporal.omes.kitchen_sink.HandlerInvocation.prototype.setArgsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 2, value);
};


/**
 * @param {!proto.temporal.api.common.v1.Payload=} opt_value
 * @param {number=} opt_index
 * @return {!proto.temporal.api.common.v1.Payload}
 */
proto.temporal.omes.kitchen_sink.HandlerInvocation.prototype.addArgs = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 2, opt_value, proto.temporal.api.common.v1.Payload, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.temporal.omes.kitchen_sink.HandlerInvocation} returns this
 */
proto.temporal.omes.kitchen_sink.HandlerInvocation.prototype.clearArgsList = function() {
  return this.setArgsList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.WorkflowState.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.WorkflowState.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.WorkflowState} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.WorkflowState.toObject = function(includeInstance, msg) {
  var f, obj = {
kvsMap: (f = msg.getKvsMap()) ? f.toObject(includeInstance, undefined) : []
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.WorkflowState}
 */
proto.temporal.omes.kitchen_sink.WorkflowState.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.WorkflowState;
  return proto.temporal.omes.kitchen_sink.WorkflowState.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.WorkflowState} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.WorkflowState}
 */
proto.temporal.omes.kitchen_sink.WorkflowState.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = msg.getKvsMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readString, null, "", "");
         });
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.WorkflowState.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.WorkflowState.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.WorkflowState} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.WorkflowState.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getKvsMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(1, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeString);
  }
};


/**
 * map<string, string> kvs = 1;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,string>}
 */
proto.temporal.omes.kitchen_sink.WorkflowState.prototype.getKvsMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,string>} */ (
      jspb.Message.getMapField(this, 1, opt_noLazyCreate,
      null));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.temporal.omes.kitchen_sink.WorkflowState} returns this
 */
proto.temporal.omes.kitchen_sink.WorkflowState.prototype.clearKvsMap = function() {
  this.getKvsMap().clear();
  return this;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.temporal.omes.kitchen_sink.WorkflowInput.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.WorkflowInput.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.WorkflowInput.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.WorkflowInput} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.WorkflowInput.toObject = function(includeInstance, msg) {
  var f, obj = {
initialActionsList: jspb.Message.toObjectList(msg.getInitialActionsList(),
    proto.temporal.omes.kitchen_sink.ActionSet.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.WorkflowInput}
 */
proto.temporal.omes.kitchen_sink.WorkflowInput.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.WorkflowInput;
  return proto.temporal.omes.kitchen_sink.WorkflowInput.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.WorkflowInput} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.WorkflowInput}
 */
proto.temporal.omes.kitchen_sink.WorkflowInput.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.temporal.omes.kitchen_sink.ActionSet;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.ActionSet.deserializeBinaryFromReader);
      msg.addInitialActions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.WorkflowInput.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.WorkflowInput.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.WorkflowInput} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.WorkflowInput.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getInitialActionsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.temporal.omes.kitchen_sink.ActionSet.serializeBinaryToWriter
    );
  }
};


/**
 * repeated ActionSet initial_actions = 1;
 * @return {!Array<!proto.temporal.omes.kitchen_sink.ActionSet>}
 */
proto.temporal.omes.kitchen_sink.WorkflowInput.prototype.getInitialActionsList = function() {
  return /** @type{!Array<!proto.temporal.omes.kitchen_sink.ActionSet>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.temporal.omes.kitchen_sink.ActionSet, 1));
};


/**
 * @param {!Array<!proto.temporal.omes.kitchen_sink.ActionSet>} value
 * @return {!proto.temporal.omes.kitchen_sink.WorkflowInput} returns this
*/
proto.temporal.omes.kitchen_sink.WorkflowInput.prototype.setInitialActionsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.temporal.omes.kitchen_sink.ActionSet=} opt_value
 * @param {number=} opt_index
 * @return {!proto.temporal.omes.kitchen_sink.ActionSet}
 */
proto.temporal.omes.kitchen_sink.WorkflowInput.prototype.addInitialActions = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.temporal.omes.kitchen_sink.ActionSet, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.temporal.omes.kitchen_sink.WorkflowInput} returns this
 */
proto.temporal.omes.kitchen_sink.WorkflowInput.prototype.clearInitialActionsList = function() {
  return this.setInitialActionsList([]);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.temporal.omes.kitchen_sink.ActionSet.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.ActionSet.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.ActionSet.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.ActionSet} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.ActionSet.toObject = function(includeInstance, msg) {
  var f, obj = {
actionsList: jspb.Message.toObjectList(msg.getActionsList(),
    proto.temporal.omes.kitchen_sink.Action.toObject, includeInstance),
concurrent: jspb.Message.getBooleanFieldWithDefault(msg, 2, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.ActionSet}
 */
proto.temporal.omes.kitchen_sink.ActionSet.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.ActionSet;
  return proto.temporal.omes.kitchen_sink.ActionSet.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.ActionSet} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.ActionSet}
 */
proto.temporal.omes.kitchen_sink.ActionSet.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.temporal.omes.kitchen_sink.Action;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.Action.deserializeBinaryFromReader);
      msg.addActions(value);
      break;
    case 2:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setConcurrent(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.ActionSet.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.ActionSet.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.ActionSet} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.ActionSet.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getActionsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.temporal.omes.kitchen_sink.Action.serializeBinaryToWriter
    );
  }
  f = message.getConcurrent();
  if (f) {
    writer.writeBool(
      2,
      f
    );
  }
};


/**
 * repeated Action actions = 1;
 * @return {!Array<!proto.temporal.omes.kitchen_sink.Action>}
 */
proto.temporal.omes.kitchen_sink.ActionSet.prototype.getActionsList = function() {
  return /** @type{!Array<!proto.temporal.omes.kitchen_sink.Action>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.temporal.omes.kitchen_sink.Action, 1));
};


/**
 * @param {!Array<!proto.temporal.omes.kitchen_sink.Action>} value
 * @return {!proto.temporal.omes.kitchen_sink.ActionSet} returns this
*/
proto.temporal.omes.kitchen_sink.ActionSet.prototype.setActionsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.temporal.omes.kitchen_sink.Action=} opt_value
 * @param {number=} opt_index
 * @return {!proto.temporal.omes.kitchen_sink.Action}
 */
proto.temporal.omes.kitchen_sink.ActionSet.prototype.addActions = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.temporal.omes.kitchen_sink.Action, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.temporal.omes.kitchen_sink.ActionSet} returns this
 */
proto.temporal.omes.kitchen_sink.ActionSet.prototype.clearActionsList = function() {
  return this.setActionsList([]);
};


/**
 * optional bool concurrent = 2;
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ActionSet.prototype.getConcurrent = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 2, false));
};


/**
 * @param {boolean} value
 * @return {!proto.temporal.omes.kitchen_sink.ActionSet} returns this
 */
proto.temporal.omes.kitchen_sink.ActionSet.prototype.setConcurrent = function(value) {
  return jspb.Message.setProto3BooleanField(this, 2, value);
};



/**
 * Oneof group definitions for this message. Each group defines the field
 * numbers belonging to that group. When of these fields' value is set, all
 * other fields in the group are cleared. During deserialization, if multiple
 * fields are encountered for a group, only the last value seen will be kept.
 * @private {!Array<!Array<number>>}
 * @const
 */
proto.temporal.omes.kitchen_sink.Action.oneofGroups_ = [[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15]];

/**
 * @enum {number}
 */
proto.temporal.omes.kitchen_sink.Action.VariantCase = {
  VARIANT_NOT_SET: 0,
  TIMER: 1,
  EXEC_ACTIVITY: 2,
  EXEC_CHILD_WORKFLOW: 3,
  AWAIT_WORKFLOW_STATE: 4,
  SEND_SIGNAL: 5,
  CANCEL_WORKFLOW: 6,
  SET_PATCH_MARKER: 7,
  UPSERT_SEARCH_ATTRIBUTES: 8,
  UPSERT_MEMO: 9,
  SET_WORKFLOW_STATE: 10,
  RETURN_RESULT: 11,
  RETURN_ERROR: 12,
  CONTINUE_AS_NEW: 13,
  NESTED_ACTION_SET: 14,
  NEXUS_OPERATION: 15
};

/**
 * @return {proto.temporal.omes.kitchen_sink.Action.VariantCase}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.getVariantCase = function() {
  return /** @type {proto.temporal.omes.kitchen_sink.Action.VariantCase} */(jspb.Message.computeOneofCase(this, proto.temporal.omes.kitchen_sink.Action.oneofGroups_[0]));
};



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.Action.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.Action} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.Action.toObject = function(includeInstance, msg) {
  var f, obj = {
timer: (f = msg.getTimer()) && proto.temporal.omes.kitchen_sink.TimerAction.toObject(includeInstance, f),
execActivity: (f = msg.getExecActivity()) && proto.temporal.omes.kitchen_sink.ExecuteActivityAction.toObject(includeInstance, f),
execChildWorkflow: (f = msg.getExecChildWorkflow()) && proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.toObject(includeInstance, f),
awaitWorkflowState: (f = msg.getAwaitWorkflowState()) && proto.temporal.omes.kitchen_sink.AwaitWorkflowState.toObject(includeInstance, f),
sendSignal: (f = msg.getSendSignal()) && proto.temporal.omes.kitchen_sink.SendSignalAction.toObject(includeInstance, f),
cancelWorkflow: (f = msg.getCancelWorkflow()) && proto.temporal.omes.kitchen_sink.CancelWorkflowAction.toObject(includeInstance, f),
setPatchMarker: (f = msg.getSetPatchMarker()) && proto.temporal.omes.kitchen_sink.SetPatchMarkerAction.toObject(includeInstance, f),
upsertSearchAttributes: (f = msg.getUpsertSearchAttributes()) && proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction.toObject(includeInstance, f),
upsertMemo: (f = msg.getUpsertMemo()) && proto.temporal.omes.kitchen_sink.UpsertMemoAction.toObject(includeInstance, f),
setWorkflowState: (f = msg.getSetWorkflowState()) && proto.temporal.omes.kitchen_sink.WorkflowState.toObject(includeInstance, f),
returnResult: (f = msg.getReturnResult()) && proto.temporal.omes.kitchen_sink.ReturnResultAction.toObject(includeInstance, f),
returnError: (f = msg.getReturnError()) && proto.temporal.omes.kitchen_sink.ReturnErrorAction.toObject(includeInstance, f),
continueAsNew: (f = msg.getContinueAsNew()) && proto.temporal.omes.kitchen_sink.ContinueAsNewAction.toObject(includeInstance, f),
nestedActionSet: (f = msg.getNestedActionSet()) && proto.temporal.omes.kitchen_sink.ActionSet.toObject(includeInstance, f),
nexusOperation: (f = msg.getNexusOperation()) && proto.temporal.omes.kitchen_sink.ExecuteNexusOperation.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.Action}
 */
proto.temporal.omes.kitchen_sink.Action.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.Action;
  return proto.temporal.omes.kitchen_sink.Action.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.Action} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.Action}
 */
proto.temporal.omes.kitchen_sink.Action.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.temporal.omes.kitchen_sink.TimerAction;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.TimerAction.deserializeBinaryFromReader);
      msg.setTimer(value);
      break;
    case 2:
      var value = new proto.temporal.omes.kitchen_sink.ExecuteActivityAction;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.ExecuteActivityAction.deserializeBinaryFromReader);
      msg.setExecActivity(value);
      break;
    case 3:
      var value = new proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.deserializeBinaryFromReader);
      msg.setExecChildWorkflow(value);
      break;
    case 4:
      var value = new proto.temporal.omes.kitchen_sink.AwaitWorkflowState;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.AwaitWorkflowState.deserializeBinaryFromReader);
      msg.setAwaitWorkflowState(value);
      break;
    case 5:
      var value = new proto.temporal.omes.kitchen_sink.SendSignalAction;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.SendSignalAction.deserializeBinaryFromReader);
      msg.setSendSignal(value);
      break;
    case 6:
      var value = new proto.temporal.omes.kitchen_sink.CancelWorkflowAction;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.CancelWorkflowAction.deserializeBinaryFromReader);
      msg.setCancelWorkflow(value);
      break;
    case 7:
      var value = new proto.temporal.omes.kitchen_sink.SetPatchMarkerAction;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.SetPatchMarkerAction.deserializeBinaryFromReader);
      msg.setSetPatchMarker(value);
      break;
    case 8:
      var value = new proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction.deserializeBinaryFromReader);
      msg.setUpsertSearchAttributes(value);
      break;
    case 9:
      var value = new proto.temporal.omes.kitchen_sink.UpsertMemoAction;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.UpsertMemoAction.deserializeBinaryFromReader);
      msg.setUpsertMemo(value);
      break;
    case 10:
      var value = new proto.temporal.omes.kitchen_sink.WorkflowState;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.WorkflowState.deserializeBinaryFromReader);
      msg.setSetWorkflowState(value);
      break;
    case 11:
      var value = new proto.temporal.omes.kitchen_sink.ReturnResultAction;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.ReturnResultAction.deserializeBinaryFromReader);
      msg.setReturnResult(value);
      break;
    case 12:
      var value = new proto.temporal.omes.kitchen_sink.ReturnErrorAction;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.ReturnErrorAction.deserializeBinaryFromReader);
      msg.setReturnError(value);
      break;
    case 13:
      var value = new proto.temporal.omes.kitchen_sink.ContinueAsNewAction;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.ContinueAsNewAction.deserializeBinaryFromReader);
      msg.setContinueAsNew(value);
      break;
    case 14:
      var value = new proto.temporal.omes.kitchen_sink.ActionSet;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.ActionSet.deserializeBinaryFromReader);
      msg.setNestedActionSet(value);
      break;
    case 15:
      var value = new proto.temporal.omes.kitchen_sink.ExecuteNexusOperation;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.ExecuteNexusOperation.deserializeBinaryFromReader);
      msg.setNexusOperation(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.Action.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.Action} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.Action.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getTimer();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.temporal.omes.kitchen_sink.TimerAction.serializeBinaryToWriter
    );
  }
  f = message.getExecActivity();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.temporal.omes.kitchen_sink.ExecuteActivityAction.serializeBinaryToWriter
    );
  }
  f = message.getExecChildWorkflow();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.serializeBinaryToWriter
    );
  }
  f = message.getAwaitWorkflowState();
  if (f != null) {
    writer.writeMessage(
      4,
      f,
      proto.temporal.omes.kitchen_sink.AwaitWorkflowState.serializeBinaryToWriter
    );
  }
  f = message.getSendSignal();
  if (f != null) {
    writer.writeMessage(
      5,
      f,
      proto.temporal.omes.kitchen_sink.SendSignalAction.serializeBinaryToWriter
    );
  }
  f = message.getCancelWorkflow();
  if (f != null) {
    writer.writeMessage(
      6,
      f,
      proto.temporal.omes.kitchen_sink.CancelWorkflowAction.serializeBinaryToWriter
    );
  }
  f = message.getSetPatchMarker();
  if (f != null) {
    writer.writeMessage(
      7,
      f,
      proto.temporal.omes.kitchen_sink.SetPatchMarkerAction.serializeBinaryToWriter
    );
  }
  f = message.getUpsertSearchAttributes();
  if (f != null) {
    writer.writeMessage(
      8,
      f,
      proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction.serializeBinaryToWriter
    );
  }
  f = message.getUpsertMemo();
  if (f != null) {
    writer.writeMessage(
      9,
      f,
      proto.temporal.omes.kitchen_sink.UpsertMemoAction.serializeBinaryToWriter
    );
  }
  f = message.getSetWorkflowState();
  if (f != null) {
    writer.writeMessage(
      10,
      f,
      proto.temporal.omes.kitchen_sink.WorkflowState.serializeBinaryToWriter
    );
  }
  f = message.getReturnResult();
  if (f != null) {
    writer.writeMessage(
      11,
      f,
      proto.temporal.omes.kitchen_sink.ReturnResultAction.serializeBinaryToWriter
    );
  }
  f = message.getReturnError();
  if (f != null) {
    writer.writeMessage(
      12,
      f,
      proto.temporal.omes.kitchen_sink.ReturnErrorAction.serializeBinaryToWriter
    );
  }
  f = message.getContinueAsNew();
  if (f != null) {
    writer.writeMessage(
      13,
      f,
      proto.temporal.omes.kitchen_sink.ContinueAsNewAction.serializeBinaryToWriter
    );
  }
  f = message.getNestedActionSet();
  if (f != null) {
    writer.writeMessage(
      14,
      f,
      proto.temporal.omes.kitchen_sink.ActionSet.serializeBinaryToWriter
    );
  }
  f = message.getNexusOperation();
  if (f != null) {
    writer.writeMessage(
      15,
      f,
      proto.temporal.omes.kitchen_sink.ExecuteNexusOperation.serializeBinaryToWriter
    );
  }
};


/**
 * optional TimerAction timer = 1;
 * @return {?proto.temporal.omes.kitchen_sink.TimerAction}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.getTimer = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.TimerAction} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.TimerAction, 1));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.TimerAction|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.Action} returns this
*/
proto.temporal.omes.kitchen_sink.Action.prototype.setTimer = function(value) {
  return jspb.Message.setOneofWrapperField(this, 1, proto.temporal.omes.kitchen_sink.Action.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.Action} returns this
 */
proto.temporal.omes.kitchen_sink.Action.prototype.clearTimer = function() {
  return this.setTimer(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.hasTimer = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional ExecuteActivityAction exec_activity = 2;
 * @return {?proto.temporal.omes.kitchen_sink.ExecuteActivityAction}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.getExecActivity = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.ExecuteActivityAction} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.ExecuteActivityAction, 2));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.ExecuteActivityAction|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.Action} returns this
*/
proto.temporal.omes.kitchen_sink.Action.prototype.setExecActivity = function(value) {
  return jspb.Message.setOneofWrapperField(this, 2, proto.temporal.omes.kitchen_sink.Action.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.Action} returns this
 */
proto.temporal.omes.kitchen_sink.Action.prototype.clearExecActivity = function() {
  return this.setExecActivity(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.hasExecActivity = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional ExecuteChildWorkflowAction exec_child_workflow = 3;
 * @return {?proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.getExecChildWorkflow = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction, 3));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.Action} returns this
*/
proto.temporal.omes.kitchen_sink.Action.prototype.setExecChildWorkflow = function(value) {
  return jspb.Message.setOneofWrapperField(this, 3, proto.temporal.omes.kitchen_sink.Action.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.Action} returns this
 */
proto.temporal.omes.kitchen_sink.Action.prototype.clearExecChildWorkflow = function() {
  return this.setExecChildWorkflow(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.hasExecChildWorkflow = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional AwaitWorkflowState await_workflow_state = 4;
 * @return {?proto.temporal.omes.kitchen_sink.AwaitWorkflowState}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.getAwaitWorkflowState = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.AwaitWorkflowState} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.AwaitWorkflowState, 4));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.AwaitWorkflowState|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.Action} returns this
*/
proto.temporal.omes.kitchen_sink.Action.prototype.setAwaitWorkflowState = function(value) {
  return jspb.Message.setOneofWrapperField(this, 4, proto.temporal.omes.kitchen_sink.Action.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.Action} returns this
 */
proto.temporal.omes.kitchen_sink.Action.prototype.clearAwaitWorkflowState = function() {
  return this.setAwaitWorkflowState(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.hasAwaitWorkflowState = function() {
  return jspb.Message.getField(this, 4) != null;
};


/**
 * optional SendSignalAction send_signal = 5;
 * @return {?proto.temporal.omes.kitchen_sink.SendSignalAction}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.getSendSignal = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.SendSignalAction} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.SendSignalAction, 5));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.SendSignalAction|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.Action} returns this
*/
proto.temporal.omes.kitchen_sink.Action.prototype.setSendSignal = function(value) {
  return jspb.Message.setOneofWrapperField(this, 5, proto.temporal.omes.kitchen_sink.Action.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.Action} returns this
 */
proto.temporal.omes.kitchen_sink.Action.prototype.clearSendSignal = function() {
  return this.setSendSignal(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.hasSendSignal = function() {
  return jspb.Message.getField(this, 5) != null;
};


/**
 * optional CancelWorkflowAction cancel_workflow = 6;
 * @return {?proto.temporal.omes.kitchen_sink.CancelWorkflowAction}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.getCancelWorkflow = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.CancelWorkflowAction} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.CancelWorkflowAction, 6));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.CancelWorkflowAction|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.Action} returns this
*/
proto.temporal.omes.kitchen_sink.Action.prototype.setCancelWorkflow = function(value) {
  return jspb.Message.setOneofWrapperField(this, 6, proto.temporal.omes.kitchen_sink.Action.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.Action} returns this
 */
proto.temporal.omes.kitchen_sink.Action.prototype.clearCancelWorkflow = function() {
  return this.setCancelWorkflow(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.hasCancelWorkflow = function() {
  return jspb.Message.getField(this, 6) != null;
};


/**
 * optional SetPatchMarkerAction set_patch_marker = 7;
 * @return {?proto.temporal.omes.kitchen_sink.SetPatchMarkerAction}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.getSetPatchMarker = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.SetPatchMarkerAction} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.SetPatchMarkerAction, 7));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.SetPatchMarkerAction|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.Action} returns this
*/
proto.temporal.omes.kitchen_sink.Action.prototype.setSetPatchMarker = function(value) {
  return jspb.Message.setOneofWrapperField(this, 7, proto.temporal.omes.kitchen_sink.Action.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.Action} returns this
 */
proto.temporal.omes.kitchen_sink.Action.prototype.clearSetPatchMarker = function() {
  return this.setSetPatchMarker(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.hasSetPatchMarker = function() {
  return jspb.Message.getField(this, 7) != null;
};


/**
 * optional UpsertSearchAttributesAction upsert_search_attributes = 8;
 * @return {?proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.getUpsertSearchAttributes = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction, 8));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.Action} returns this
*/
proto.temporal.omes.kitchen_sink.Action.prototype.setUpsertSearchAttributes = function(value) {
  return jspb.Message.setOneofWrapperField(this, 8, proto.temporal.omes.kitchen_sink.Action.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.Action} returns this
 */
proto.temporal.omes.kitchen_sink.Action.prototype.clearUpsertSearchAttributes = function() {
  return this.setUpsertSearchAttributes(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.hasUpsertSearchAttributes = function() {
  return jspb.Message.getField(this, 8) != null;
};


/**
 * optional UpsertMemoAction upsert_memo = 9;
 * @return {?proto.temporal.omes.kitchen_sink.UpsertMemoAction}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.getUpsertMemo = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.UpsertMemoAction} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.UpsertMemoAction, 9));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.UpsertMemoAction|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.Action} returns this
*/
proto.temporal.omes.kitchen_sink.Action.prototype.setUpsertMemo = function(value) {
  return jspb.Message.setOneofWrapperField(this, 9, proto.temporal.omes.kitchen_sink.Action.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.Action} returns this
 */
proto.temporal.omes.kitchen_sink.Action.prototype.clearUpsertMemo = function() {
  return this.setUpsertMemo(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.hasUpsertMemo = function() {
  return jspb.Message.getField(this, 9) != null;
};


/**
 * optional WorkflowState set_workflow_state = 10;
 * @return {?proto.temporal.omes.kitchen_sink.WorkflowState}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.getSetWorkflowState = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.WorkflowState} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.WorkflowState, 10));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.WorkflowState|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.Action} returns this
*/
proto.temporal.omes.kitchen_sink.Action.prototype.setSetWorkflowState = function(value) {
  return jspb.Message.setOneofWrapperField(this, 10, proto.temporal.omes.kitchen_sink.Action.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.Action} returns this
 */
proto.temporal.omes.kitchen_sink.Action.prototype.clearSetWorkflowState = function() {
  return this.setSetWorkflowState(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.hasSetWorkflowState = function() {
  return jspb.Message.getField(this, 10) != null;
};


/**
 * optional ReturnResultAction return_result = 11;
 * @return {?proto.temporal.omes.kitchen_sink.ReturnResultAction}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.getReturnResult = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.ReturnResultAction} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.ReturnResultAction, 11));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.ReturnResultAction|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.Action} returns this
*/
proto.temporal.omes.kitchen_sink.Action.prototype.setReturnResult = function(value) {
  return jspb.Message.setOneofWrapperField(this, 11, proto.temporal.omes.kitchen_sink.Action.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.Action} returns this
 */
proto.temporal.omes.kitchen_sink.Action.prototype.clearReturnResult = function() {
  return this.setReturnResult(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.hasReturnResult = function() {
  return jspb.Message.getField(this, 11) != null;
};


/**
 * optional ReturnErrorAction return_error = 12;
 * @return {?proto.temporal.omes.kitchen_sink.ReturnErrorAction}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.getReturnError = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.ReturnErrorAction} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.ReturnErrorAction, 12));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.ReturnErrorAction|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.Action} returns this
*/
proto.temporal.omes.kitchen_sink.Action.prototype.setReturnError = function(value) {
  return jspb.Message.setOneofWrapperField(this, 12, proto.temporal.omes.kitchen_sink.Action.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.Action} returns this
 */
proto.temporal.omes.kitchen_sink.Action.prototype.clearReturnError = function() {
  return this.setReturnError(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.hasReturnError = function() {
  return jspb.Message.getField(this, 12) != null;
};


/**
 * optional ContinueAsNewAction continue_as_new = 13;
 * @return {?proto.temporal.omes.kitchen_sink.ContinueAsNewAction}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.getContinueAsNew = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.ContinueAsNewAction} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.ContinueAsNewAction, 13));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.ContinueAsNewAction|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.Action} returns this
*/
proto.temporal.omes.kitchen_sink.Action.prototype.setContinueAsNew = function(value) {
  return jspb.Message.setOneofWrapperField(this, 13, proto.temporal.omes.kitchen_sink.Action.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.Action} returns this
 */
proto.temporal.omes.kitchen_sink.Action.prototype.clearContinueAsNew = function() {
  return this.setContinueAsNew(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.hasContinueAsNew = function() {
  return jspb.Message.getField(this, 13) != null;
};


/**
 * optional ActionSet nested_action_set = 14;
 * @return {?proto.temporal.omes.kitchen_sink.ActionSet}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.getNestedActionSet = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.ActionSet} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.ActionSet, 14));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.ActionSet|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.Action} returns this
*/
proto.temporal.omes.kitchen_sink.Action.prototype.setNestedActionSet = function(value) {
  return jspb.Message.setOneofWrapperField(this, 14, proto.temporal.omes.kitchen_sink.Action.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.Action} returns this
 */
proto.temporal.omes.kitchen_sink.Action.prototype.clearNestedActionSet = function() {
  return this.setNestedActionSet(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.hasNestedActionSet = function() {
  return jspb.Message.getField(this, 14) != null;
};


/**
 * optional ExecuteNexusOperation nexus_operation = 15;
 * @return {?proto.temporal.omes.kitchen_sink.ExecuteNexusOperation}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.getNexusOperation = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.ExecuteNexusOperation} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.ExecuteNexusOperation, 15));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.ExecuteNexusOperation|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.Action} returns this
*/
proto.temporal.omes.kitchen_sink.Action.prototype.setNexusOperation = function(value) {
  return jspb.Message.setOneofWrapperField(this, 15, proto.temporal.omes.kitchen_sink.Action.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.Action} returns this
 */
proto.temporal.omes.kitchen_sink.Action.prototype.clearNexusOperation = function() {
  return this.setNexusOperation(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.Action.prototype.hasNexusOperation = function() {
  return jspb.Message.getField(this, 15) != null;
};



/**
 * Oneof group definitions for this message. Each group defines the field
 * numbers belonging to that group. When of these fields' value is set, all
 * other fields in the group are cleared. During deserialization, if multiple
 * fields are encountered for a group, only the last value seen will be kept.
 * @private {!Array<!Array<number>>}
 * @const
 */
proto.temporal.omes.kitchen_sink.AwaitableChoice.oneofGroups_ = [[1,2,3,4,5]];

/**
 * @enum {number}
 */
proto.temporal.omes.kitchen_sink.AwaitableChoice.ConditionCase = {
  CONDITION_NOT_SET: 0,
  WAIT_FINISH: 1,
  ABANDON: 2,
  CANCEL_BEFORE_STARTED: 3,
  CANCEL_AFTER_STARTED: 4,
  CANCEL_AFTER_COMPLETED: 5
};

/**
 * @return {proto.temporal.omes.kitchen_sink.AwaitableChoice.ConditionCase}
 */
proto.temporal.omes.kitchen_sink.AwaitableChoice.prototype.getConditionCase = function() {
  return /** @type {proto.temporal.omes.kitchen_sink.AwaitableChoice.ConditionCase} */(jspb.Message.computeOneofCase(this, proto.temporal.omes.kitchen_sink.AwaitableChoice.oneofGroups_[0]));
};



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.AwaitableChoice.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.AwaitableChoice.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.AwaitableChoice} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.AwaitableChoice.toObject = function(includeInstance, msg) {
  var f, obj = {
waitFinish: (f = msg.getWaitFinish()) && google_protobuf_empty_pb.Empty.toObject(includeInstance, f),
abandon: (f = msg.getAbandon()) && google_protobuf_empty_pb.Empty.toObject(includeInstance, f),
cancelBeforeStarted: (f = msg.getCancelBeforeStarted()) && google_protobuf_empty_pb.Empty.toObject(includeInstance, f),
cancelAfterStarted: (f = msg.getCancelAfterStarted()) && google_protobuf_empty_pb.Empty.toObject(includeInstance, f),
cancelAfterCompleted: (f = msg.getCancelAfterCompleted()) && google_protobuf_empty_pb.Empty.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.AwaitableChoice}
 */
proto.temporal.omes.kitchen_sink.AwaitableChoice.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.AwaitableChoice;
  return proto.temporal.omes.kitchen_sink.AwaitableChoice.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.AwaitableChoice} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.AwaitableChoice}
 */
proto.temporal.omes.kitchen_sink.AwaitableChoice.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new google_protobuf_empty_pb.Empty;
      reader.readMessage(value,google_protobuf_empty_pb.Empty.deserializeBinaryFromReader);
      msg.setWaitFinish(value);
      break;
    case 2:
      var value = new google_protobuf_empty_pb.Empty;
      reader.readMessage(value,google_protobuf_empty_pb.Empty.deserializeBinaryFromReader);
      msg.setAbandon(value);
      break;
    case 3:
      var value = new google_protobuf_empty_pb.Empty;
      reader.readMessage(value,google_protobuf_empty_pb.Empty.deserializeBinaryFromReader);
      msg.setCancelBeforeStarted(value);
      break;
    case 4:
      var value = new google_protobuf_empty_pb.Empty;
      reader.readMessage(value,google_protobuf_empty_pb.Empty.deserializeBinaryFromReader);
      msg.setCancelAfterStarted(value);
      break;
    case 5:
      var value = new google_protobuf_empty_pb.Empty;
      reader.readMessage(value,google_protobuf_empty_pb.Empty.deserializeBinaryFromReader);
      msg.setCancelAfterCompleted(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.AwaitableChoice.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.AwaitableChoice.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.AwaitableChoice} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.AwaitableChoice.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getWaitFinish();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      google_protobuf_empty_pb.Empty.serializeBinaryToWriter
    );
  }
  f = message.getAbandon();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      google_protobuf_empty_pb.Empty.serializeBinaryToWriter
    );
  }
  f = message.getCancelBeforeStarted();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      google_protobuf_empty_pb.Empty.serializeBinaryToWriter
    );
  }
  f = message.getCancelAfterStarted();
  if (f != null) {
    writer.writeMessage(
      4,
      f,
      google_protobuf_empty_pb.Empty.serializeBinaryToWriter
    );
  }
  f = message.getCancelAfterCompleted();
  if (f != null) {
    writer.writeMessage(
      5,
      f,
      google_protobuf_empty_pb.Empty.serializeBinaryToWriter
    );
  }
};


/**
 * optional google.protobuf.Empty wait_finish = 1;
 * @return {?proto.google.protobuf.Empty}
 */
proto.temporal.omes.kitchen_sink.AwaitableChoice.prototype.getWaitFinish = function() {
  return /** @type{?proto.google.protobuf.Empty} */ (
    jspb.Message.getWrapperField(this, google_protobuf_empty_pb.Empty, 1));
};


/**
 * @param {?proto.google.protobuf.Empty|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.AwaitableChoice} returns this
*/
proto.temporal.omes.kitchen_sink.AwaitableChoice.prototype.setWaitFinish = function(value) {
  return jspb.Message.setOneofWrapperField(this, 1, proto.temporal.omes.kitchen_sink.AwaitableChoice.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.AwaitableChoice} returns this
 */
proto.temporal.omes.kitchen_sink.AwaitableChoice.prototype.clearWaitFinish = function() {
  return this.setWaitFinish(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.AwaitableChoice.prototype.hasWaitFinish = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional google.protobuf.Empty abandon = 2;
 * @return {?proto.google.protobuf.Empty}
 */
proto.temporal.omes.kitchen_sink.AwaitableChoice.prototype.getAbandon = function() {
  return /** @type{?proto.google.protobuf.Empty} */ (
    jspb.Message.getWrapperField(this, google_protobuf_empty_pb.Empty, 2));
};


/**
 * @param {?proto.google.protobuf.Empty|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.AwaitableChoice} returns this
*/
proto.temporal.omes.kitchen_sink.AwaitableChoice.prototype.setAbandon = function(value) {
  return jspb.Message.setOneofWrapperField(this, 2, proto.temporal.omes.kitchen_sink.AwaitableChoice.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.AwaitableChoice} returns this
 */
proto.temporal.omes.kitchen_sink.AwaitableChoice.prototype.clearAbandon = function() {
  return this.setAbandon(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.AwaitableChoice.prototype.hasAbandon = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional google.protobuf.Empty cancel_before_started = 3;
 * @return {?proto.google.protobuf.Empty}
 */
proto.temporal.omes.kitchen_sink.AwaitableChoice.prototype.getCancelBeforeStarted = function() {
  return /** @type{?proto.google.protobuf.Empty} */ (
    jspb.Message.getWrapperField(this, google_protobuf_empty_pb.Empty, 3));
};


/**
 * @param {?proto.google.protobuf.Empty|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.AwaitableChoice} returns this
*/
proto.temporal.omes.kitchen_sink.AwaitableChoice.prototype.setCancelBeforeStarted = function(value) {
  return jspb.Message.setOneofWrapperField(this, 3, proto.temporal.omes.kitchen_sink.AwaitableChoice.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.AwaitableChoice} returns this
 */
proto.temporal.omes.kitchen_sink.AwaitableChoice.prototype.clearCancelBeforeStarted = function() {
  return this.setCancelBeforeStarted(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.AwaitableChoice.prototype.hasCancelBeforeStarted = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional google.protobuf.Empty cancel_after_started = 4;
 * @return {?proto.google.protobuf.Empty}
 */
proto.temporal.omes.kitchen_sink.AwaitableChoice.prototype.getCancelAfterStarted = function() {
  return /** @type{?proto.google.protobuf.Empty} */ (
    jspb.Message.getWrapperField(this, google_protobuf_empty_pb.Empty, 4));
};


/**
 * @param {?proto.google.protobuf.Empty|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.AwaitableChoice} returns this
*/
proto.temporal.omes.kitchen_sink.AwaitableChoice.prototype.setCancelAfterStarted = function(value) {
  return jspb.Message.setOneofWrapperField(this, 4, proto.temporal.omes.kitchen_sink.AwaitableChoice.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.AwaitableChoice} returns this
 */
proto.temporal.omes.kitchen_sink.AwaitableChoice.prototype.clearCancelAfterStarted = function() {
  return this.setCancelAfterStarted(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.AwaitableChoice.prototype.hasCancelAfterStarted = function() {
  return jspb.Message.getField(this, 4) != null;
};


/**
 * optional google.protobuf.Empty cancel_after_completed = 5;
 * @return {?proto.google.protobuf.Empty}
 */
proto.temporal.omes.kitchen_sink.AwaitableChoice.prototype.getCancelAfterCompleted = function() {
  return /** @type{?proto.google.protobuf.Empty} */ (
    jspb.Message.getWrapperField(this, google_protobuf_empty_pb.Empty, 5));
};


/**
 * @param {?proto.google.protobuf.Empty|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.AwaitableChoice} returns this
*/
proto.temporal.omes.kitchen_sink.AwaitableChoice.prototype.setCancelAfterCompleted = function(value) {
  return jspb.Message.setOneofWrapperField(this, 5, proto.temporal.omes.kitchen_sink.AwaitableChoice.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.AwaitableChoice} returns this
 */
proto.temporal.omes.kitchen_sink.AwaitableChoice.prototype.clearCancelAfterCompleted = function() {
  return this.setCancelAfterCompleted(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.AwaitableChoice.prototype.hasCancelAfterCompleted = function() {
  return jspb.Message.getField(this, 5) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.TimerAction.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.TimerAction.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.TimerAction} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.TimerAction.toObject = function(includeInstance, msg) {
  var f, obj = {
milliseconds: jspb.Message.getFieldWithDefault(msg, 1, 0),
awaitableChoice: (f = msg.getAwaitableChoice()) && proto.temporal.omes.kitchen_sink.AwaitableChoice.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.TimerAction}
 */
proto.temporal.omes.kitchen_sink.TimerAction.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.TimerAction;
  return proto.temporal.omes.kitchen_sink.TimerAction.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.TimerAction} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.TimerAction}
 */
proto.temporal.omes.kitchen_sink.TimerAction.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readUint64());
      msg.setMilliseconds(value);
      break;
    case 2:
      var value = new proto.temporal.omes.kitchen_sink.AwaitableChoice;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.AwaitableChoice.deserializeBinaryFromReader);
      msg.setAwaitableChoice(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.TimerAction.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.TimerAction.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.TimerAction} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.TimerAction.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getMilliseconds();
  if (f !== 0) {
    writer.writeUint64(
      1,
      f
    );
  }
  f = message.getAwaitableChoice();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.temporal.omes.kitchen_sink.AwaitableChoice.serializeBinaryToWriter
    );
  }
};


/**
 * optional uint64 milliseconds = 1;
 * @return {number}
 */
proto.temporal.omes.kitchen_sink.TimerAction.prototype.getMilliseconds = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.temporal.omes.kitchen_sink.TimerAction} returns this
 */
proto.temporal.omes.kitchen_sink.TimerAction.prototype.setMilliseconds = function(value) {
  return jspb.Message.setProto3IntField(this, 1, value);
};


/**
 * optional AwaitableChoice awaitable_choice = 2;
 * @return {?proto.temporal.omes.kitchen_sink.AwaitableChoice}
 */
proto.temporal.omes.kitchen_sink.TimerAction.prototype.getAwaitableChoice = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.AwaitableChoice} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.AwaitableChoice, 2));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.AwaitableChoice|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.TimerAction} returns this
*/
proto.temporal.omes.kitchen_sink.TimerAction.prototype.setAwaitableChoice = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.TimerAction} returns this
 */
proto.temporal.omes.kitchen_sink.TimerAction.prototype.clearAwaitableChoice = function() {
  return this.setAwaitableChoice(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.TimerAction.prototype.hasAwaitableChoice = function() {
  return jspb.Message.getField(this, 2) != null;
};



/**
 * Oneof group definitions for this message. Each group defines the field
 * numbers belonging to that group. When of these fields' value is set, all
 * other fields in the group are cleared. During deserialization, if multiple
 * fields are encountered for a group, only the last value seen will be kept.
 * @private {!Array<!Array<number>>}
 * @const
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.oneofGroups_ = [[1,2,3,14,18],[11,12]];

/**
 * @enum {number}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ActivityTypeCase = {
  ACTIVITY_TYPE_NOT_SET: 0,
  GENERIC: 1,
  DELAY: 2,
  NOOP: 3,
  RESOURCES: 14,
  PAYLOAD: 18
};

/**
 * @return {proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ActivityTypeCase}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.getActivityTypeCase = function() {
  return /** @type {proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ActivityTypeCase} */(jspb.Message.computeOneofCase(this, proto.temporal.omes.kitchen_sink.ExecuteActivityAction.oneofGroups_[0]));
};

/**
 * @enum {number}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.LocalityCase = {
  LOCALITY_NOT_SET: 0,
  IS_LOCAL: 11,
  REMOTE: 12
};

/**
 * @return {proto.temporal.omes.kitchen_sink.ExecuteActivityAction.LocalityCase}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.getLocalityCase = function() {
  return /** @type {proto.temporal.omes.kitchen_sink.ExecuteActivityAction.LocalityCase} */(jspb.Message.computeOneofCase(this, proto.temporal.omes.kitchen_sink.ExecuteActivityAction.oneofGroups_[1]));
};



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.ExecuteActivityAction.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.toObject = function(includeInstance, msg) {
  var f, obj = {
generic: (f = msg.getGeneric()) && proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity.toObject(includeInstance, f),
delay: (f = msg.getDelay()) && google_protobuf_duration_pb.Duration.toObject(includeInstance, f),
noop: (f = msg.getNoop()) && google_protobuf_empty_pb.Empty.toObject(includeInstance, f),
resources: (f = msg.getResources()) && proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity.toObject(includeInstance, f),
payload: (f = msg.getPayload()) && proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity.toObject(includeInstance, f),
taskQueue: jspb.Message.getFieldWithDefault(msg, 4, ""),
headersMap: (f = msg.getHeadersMap()) ? f.toObject(includeInstance, proto.temporal.api.common.v1.Payload.toObject) : [],
scheduleToCloseTimeout: (f = msg.getScheduleToCloseTimeout()) && google_protobuf_duration_pb.Duration.toObject(includeInstance, f),
scheduleToStartTimeout: (f = msg.getScheduleToStartTimeout()) && google_protobuf_duration_pb.Duration.toObject(includeInstance, f),
startToCloseTimeout: (f = msg.getStartToCloseTimeout()) && google_protobuf_duration_pb.Duration.toObject(includeInstance, f),
heartbeatTimeout: (f = msg.getHeartbeatTimeout()) && google_protobuf_duration_pb.Duration.toObject(includeInstance, f),
retryPolicy: (f = msg.getRetryPolicy()) && temporal_api_common_v1_message_pb.RetryPolicy.toObject(includeInstance, f),
isLocal: (f = msg.getIsLocal()) && google_protobuf_empty_pb.Empty.toObject(includeInstance, f),
remote: (f = msg.getRemote()) && proto.temporal.omes.kitchen_sink.RemoteActivityOptions.toObject(includeInstance, f),
awaitableChoice: (f = msg.getAwaitableChoice()) && proto.temporal.omes.kitchen_sink.AwaitableChoice.toObject(includeInstance, f),
priority: (f = msg.getPriority()) && temporal_api_common_v1_message_pb.Priority.toObject(includeInstance, f),
fairnessKey: jspb.Message.getFieldWithDefault(msg, 16, ""),
fairnessWeight: jspb.Message.getFloatingPointFieldWithDefault(msg, 17, 0.0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.ExecuteActivityAction;
  return proto.temporal.omes.kitchen_sink.ExecuteActivityAction.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity.deserializeBinaryFromReader);
      msg.setGeneric(value);
      break;
    case 2:
      var value = new google_protobuf_duration_pb.Duration;
      reader.readMessage(value,google_protobuf_duration_pb.Duration.deserializeBinaryFromReader);
      msg.setDelay(value);
      break;
    case 3:
      var value = new google_protobuf_empty_pb.Empty;
      reader.readMessage(value,google_protobuf_empty_pb.Empty.deserializeBinaryFromReader);
      msg.setNoop(value);
      break;
    case 14:
      var value = new proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity.deserializeBinaryFromReader);
      msg.setResources(value);
      break;
    case 18:
      var value = new proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity.deserializeBinaryFromReader);
      msg.setPayload(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setTaskQueue(value);
      break;
    case 5:
      var value = msg.getHeadersMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.temporal.api.common.v1.Payload.deserializeBinaryFromReader, "", new proto.temporal.api.common.v1.Payload());
         });
      break;
    case 6:
      var value = new google_protobuf_duration_pb.Duration;
      reader.readMessage(value,google_protobuf_duration_pb.Duration.deserializeBinaryFromReader);
      msg.setScheduleToCloseTimeout(value);
      break;
    case 7:
      var value = new google_protobuf_duration_pb.Duration;
      reader.readMessage(value,google_protobuf_duration_pb.Duration.deserializeBinaryFromReader);
      msg.setScheduleToStartTimeout(value);
      break;
    case 8:
      var value = new google_protobuf_duration_pb.Duration;
      reader.readMessage(value,google_protobuf_duration_pb.Duration.deserializeBinaryFromReader);
      msg.setStartToCloseTimeout(value);
      break;
    case 9:
      var value = new google_protobuf_duration_pb.Duration;
      reader.readMessage(value,google_protobuf_duration_pb.Duration.deserializeBinaryFromReader);
      msg.setHeartbeatTimeout(value);
      break;
    case 10:
      var value = new temporal_api_common_v1_message_pb.RetryPolicy;
      reader.readMessage(value,temporal_api_common_v1_message_pb.RetryPolicy.deserializeBinaryFromReader);
      msg.setRetryPolicy(value);
      break;
    case 11:
      var value = new google_protobuf_empty_pb.Empty;
      reader.readMessage(value,google_protobuf_empty_pb.Empty.deserializeBinaryFromReader);
      msg.setIsLocal(value);
      break;
    case 12:
      var value = new proto.temporal.omes.kitchen_sink.RemoteActivityOptions;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.RemoteActivityOptions.deserializeBinaryFromReader);
      msg.setRemote(value);
      break;
    case 13:
      var value = new proto.temporal.omes.kitchen_sink.AwaitableChoice;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.AwaitableChoice.deserializeBinaryFromReader);
      msg.setAwaitableChoice(value);
      break;
    case 15:
      var value = new temporal_api_common_v1_message_pb.Priority;
      reader.readMessage(value,temporal_api_common_v1_message_pb.Priority.deserializeBinaryFromReader);
      msg.setPriority(value);
      break;
    case 16:
      var value = /** @type {string} */ (reader.readString());
      msg.setFairnessKey(value);
      break;
    case 17:
      var value = /** @type {number} */ (reader.readFloat());
      msg.setFairnessWeight(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.ExecuteActivityAction.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getGeneric();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity.serializeBinaryToWriter
    );
  }
  f = message.getDelay();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      google_protobuf_duration_pb.Duration.serializeBinaryToWriter
    );
  }
  f = message.getNoop();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      google_protobuf_empty_pb.Empty.serializeBinaryToWriter
    );
  }
  f = message.getResources();
  if (f != null) {
    writer.writeMessage(
      14,
      f,
      proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity.serializeBinaryToWriter
    );
  }
  f = message.getPayload();
  if (f != null) {
    writer.writeMessage(
      18,
      f,
      proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity.serializeBinaryToWriter
    );
  }
  f = message.getTaskQueue();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
  f = message.getHeadersMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(5, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.temporal.api.common.v1.Payload.serializeBinaryToWriter);
  }
  f = message.getScheduleToCloseTimeout();
  if (f != null) {
    writer.writeMessage(
      6,
      f,
      google_protobuf_duration_pb.Duration.serializeBinaryToWriter
    );
  }
  f = message.getScheduleToStartTimeout();
  if (f != null) {
    writer.writeMessage(
      7,
      f,
      google_protobuf_duration_pb.Duration.serializeBinaryToWriter
    );
  }
  f = message.getStartToCloseTimeout();
  if (f != null) {
    writer.writeMessage(
      8,
      f,
      google_protobuf_duration_pb.Duration.serializeBinaryToWriter
    );
  }
  f = message.getHeartbeatTimeout();
  if (f != null) {
    writer.writeMessage(
      9,
      f,
      google_protobuf_duration_pb.Duration.serializeBinaryToWriter
    );
  }
  f = message.getRetryPolicy();
  if (f != null) {
    writer.writeMessage(
      10,
      f,
      temporal_api_common_v1_message_pb.RetryPolicy.serializeBinaryToWriter
    );
  }
  f = message.getIsLocal();
  if (f != null) {
    writer.writeMessage(
      11,
      f,
      google_protobuf_empty_pb.Empty.serializeBinaryToWriter
    );
  }
  f = message.getRemote();
  if (f != null) {
    writer.writeMessage(
      12,
      f,
      proto.temporal.omes.kitchen_sink.RemoteActivityOptions.serializeBinaryToWriter
    );
  }
  f = message.getAwaitableChoice();
  if (f != null) {
    writer.writeMessage(
      13,
      f,
      proto.temporal.omes.kitchen_sink.AwaitableChoice.serializeBinaryToWriter
    );
  }
  f = message.getPriority();
  if (f != null) {
    writer.writeMessage(
      15,
      f,
      temporal_api_common_v1_message_pb.Priority.serializeBinaryToWriter
    );
  }
  f = message.getFairnessKey();
  if (f.length > 0) {
    writer.writeString(
      16,
      f
    );
  }
  f = message.getFairnessWeight();
  if (f !== 0.0) {
    writer.writeFloat(
      17,
      f
    );
  }
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity.repeatedFields_ = [2];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity.toObject = function(includeInstance, msg) {
  var f, obj = {
type: jspb.Message.getFieldWithDefault(msg, 1, ""),
argumentsList: jspb.Message.toObjectList(msg.getArgumentsList(),
    temporal_api_common_v1_message_pb.Payload.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity;
  return proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setType(value);
      break;
    case 2:
      var value = new temporal_api_common_v1_message_pb.Payload;
      reader.readMessage(value,temporal_api_common_v1_message_pb.Payload.deserializeBinaryFromReader);
      msg.addArguments(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getType();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getArgumentsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      2,
      f,
      temporal_api_common_v1_message_pb.Payload.serializeBinaryToWriter
    );
  }
};


/**
 * optional string type = 1;
 * @return {string}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity.prototype.getType = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity.prototype.setType = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * repeated temporal.api.common.v1.Payload arguments = 2;
 * @return {!Array<!proto.temporal.api.common.v1.Payload>}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity.prototype.getArgumentsList = function() {
  return /** @type{!Array<!proto.temporal.api.common.v1.Payload>} */ (
    jspb.Message.getRepeatedWrapperField(this, temporal_api_common_v1_message_pb.Payload, 2));
};


/**
 * @param {!Array<!proto.temporal.api.common.v1.Payload>} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity} returns this
*/
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity.prototype.setArgumentsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 2, value);
};


/**
 * @param {!proto.temporal.api.common.v1.Payload=} opt_value
 * @param {number=} opt_index
 * @return {!proto.temporal.api.common.v1.Payload}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity.prototype.addArguments = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 2, opt_value, proto.temporal.api.common.v1.Payload, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity.prototype.clearArgumentsList = function() {
  return this.setArgumentsList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity.toObject = function(includeInstance, msg) {
  var f, obj = {
runFor: (f = msg.getRunFor()) && google_protobuf_duration_pb.Duration.toObject(includeInstance, f),
bytesToAllocate: jspb.Message.getFieldWithDefault(msg, 2, 0),
cpuYieldEveryNIterations: jspb.Message.getFieldWithDefault(msg, 3, 0),
cpuYieldForMs: jspb.Message.getFieldWithDefault(msg, 4, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity;
  return proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new google_protobuf_duration_pb.Duration;
      reader.readMessage(value,google_protobuf_duration_pb.Duration.deserializeBinaryFromReader);
      msg.setRunFor(value);
      break;
    case 2:
      var value = /** @type {number} */ (reader.readUint64());
      msg.setBytesToAllocate(value);
      break;
    case 3:
      var value = /** @type {number} */ (reader.readUint32());
      msg.setCpuYieldEveryNIterations(value);
      break;
    case 4:
      var value = /** @type {number} */ (reader.readUint32());
      msg.setCpuYieldForMs(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getRunFor();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      google_protobuf_duration_pb.Duration.serializeBinaryToWriter
    );
  }
  f = message.getBytesToAllocate();
  if (f !== 0) {
    writer.writeUint64(
      2,
      f
    );
  }
  f = message.getCpuYieldEveryNIterations();
  if (f !== 0) {
    writer.writeUint32(
      3,
      f
    );
  }
  f = message.getCpuYieldForMs();
  if (f !== 0) {
    writer.writeUint32(
      4,
      f
    );
  }
};


/**
 * optional google.protobuf.Duration run_for = 1;
 * @return {?proto.google.protobuf.Duration}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity.prototype.getRunFor = function() {
  return /** @type{?proto.google.protobuf.Duration} */ (
    jspb.Message.getWrapperField(this, google_protobuf_duration_pb.Duration, 1));
};


/**
 * @param {?proto.google.protobuf.Duration|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity} returns this
*/
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity.prototype.setRunFor = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity.prototype.clearRunFor = function() {
  return this.setRunFor(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity.prototype.hasRunFor = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional uint64 bytes_to_allocate = 2;
 * @return {number}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity.prototype.getBytesToAllocate = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {number} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity.prototype.setBytesToAllocate = function(value) {
  return jspb.Message.setProto3IntField(this, 2, value);
};


/**
 * optional uint32 cpu_yield_every_n_iterations = 3;
 * @return {number}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity.prototype.getCpuYieldEveryNIterations = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {number} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity.prototype.setCpuYieldEveryNIterations = function(value) {
  return jspb.Message.setProto3IntField(this, 3, value);
};


/**
 * optional uint32 cpu_yield_for_ms = 4;
 * @return {number}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity.prototype.getCpuYieldForMs = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 4, 0));
};


/**
 * @param {number} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity.prototype.setCpuYieldForMs = function(value) {
  return jspb.Message.setProto3IntField(this, 4, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity.toObject = function(includeInstance, msg) {
  var f, obj = {
bytesToReceive: jspb.Message.getFieldWithDefault(msg, 1, 0),
bytesToReturn: jspb.Message.getFieldWithDefault(msg, 2, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity;
  return proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setBytesToReceive(value);
      break;
    case 2:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setBytesToReturn(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getBytesToReceive();
  if (f !== 0) {
    writer.writeInt32(
      1,
      f
    );
  }
  f = message.getBytesToReturn();
  if (f !== 0) {
    writer.writeInt32(
      2,
      f
    );
  }
};


/**
 * optional int32 bytes_to_receive = 1;
 * @return {number}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity.prototype.getBytesToReceive = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity.prototype.setBytesToReceive = function(value) {
  return jspb.Message.setProto3IntField(this, 1, value);
};


/**
 * optional int32 bytes_to_return = 2;
 * @return {number}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity.prototype.getBytesToReturn = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {number} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity.prototype.setBytesToReturn = function(value) {
  return jspb.Message.setProto3IntField(this, 2, value);
};


/**
 * optional GenericActivity generic = 1;
 * @return {?proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.getGeneric = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity, 1));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
*/
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.setGeneric = function(value) {
  return jspb.Message.setOneofWrapperField(this, 1, proto.temporal.omes.kitchen_sink.ExecuteActivityAction.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.clearGeneric = function() {
  return this.setGeneric(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.hasGeneric = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional google.protobuf.Duration delay = 2;
 * @return {?proto.google.protobuf.Duration}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.getDelay = function() {
  return /** @type{?proto.google.protobuf.Duration} */ (
    jspb.Message.getWrapperField(this, google_protobuf_duration_pb.Duration, 2));
};


/**
 * @param {?proto.google.protobuf.Duration|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
*/
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.setDelay = function(value) {
  return jspb.Message.setOneofWrapperField(this, 2, proto.temporal.omes.kitchen_sink.ExecuteActivityAction.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.clearDelay = function() {
  return this.setDelay(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.hasDelay = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional google.protobuf.Empty noop = 3;
 * @return {?proto.google.protobuf.Empty}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.getNoop = function() {
  return /** @type{?proto.google.protobuf.Empty} */ (
    jspb.Message.getWrapperField(this, google_protobuf_empty_pb.Empty, 3));
};


/**
 * @param {?proto.google.protobuf.Empty|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
*/
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.setNoop = function(value) {
  return jspb.Message.setOneofWrapperField(this, 3, proto.temporal.omes.kitchen_sink.ExecuteActivityAction.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.clearNoop = function() {
  return this.setNoop(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.hasNoop = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional ResourcesActivity resources = 14;
 * @return {?proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.getResources = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity, 14));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
*/
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.setResources = function(value) {
  return jspb.Message.setOneofWrapperField(this, 14, proto.temporal.omes.kitchen_sink.ExecuteActivityAction.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.clearResources = function() {
  return this.setResources(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.hasResources = function() {
  return jspb.Message.getField(this, 14) != null;
};


/**
 * optional PayloadActivity payload = 18;
 * @return {?proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.getPayload = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity, 18));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
*/
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.setPayload = function(value) {
  return jspb.Message.setOneofWrapperField(this, 18, proto.temporal.omes.kitchen_sink.ExecuteActivityAction.oneofGroups_[0], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.clearPayload = function() {
  return this.setPayload(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.hasPayload = function() {
  return jspb.Message.getField(this, 18) != null;
};


/**
 * optional string task_queue = 4;
 * @return {string}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.getTaskQueue = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.setTaskQueue = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * map<string, temporal.api.common.v1.Payload> headers = 5;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.temporal.api.common.v1.Payload>}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.getHeadersMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.temporal.api.common.v1.Payload>} */ (
      jspb.Message.getMapField(this, 5, opt_noLazyCreate,
      proto.temporal.api.common.v1.Payload));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.clearHeadersMap = function() {
  this.getHeadersMap().clear();
  return this;
};


/**
 * optional google.protobuf.Duration schedule_to_close_timeout = 6;
 * @return {?proto.google.protobuf.Duration}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.getScheduleToCloseTimeout = function() {
  return /** @type{?proto.google.protobuf.Duration} */ (
    jspb.Message.getWrapperField(this, google_protobuf_duration_pb.Duration, 6));
};


/**
 * @param {?proto.google.protobuf.Duration|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
*/
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.setScheduleToCloseTimeout = function(value) {
  return jspb.Message.setWrapperField(this, 6, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.clearScheduleToCloseTimeout = function() {
  return this.setScheduleToCloseTimeout(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.hasScheduleToCloseTimeout = function() {
  return jspb.Message.getField(this, 6) != null;
};


/**
 * optional google.protobuf.Duration schedule_to_start_timeout = 7;
 * @return {?proto.google.protobuf.Duration}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.getScheduleToStartTimeout = function() {
  return /** @type{?proto.google.protobuf.Duration} */ (
    jspb.Message.getWrapperField(this, google_protobuf_duration_pb.Duration, 7));
};


/**
 * @param {?proto.google.protobuf.Duration|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
*/
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.setScheduleToStartTimeout = function(value) {
  return jspb.Message.setWrapperField(this, 7, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.clearScheduleToStartTimeout = function() {
  return this.setScheduleToStartTimeout(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.hasScheduleToStartTimeout = function() {
  return jspb.Message.getField(this, 7) != null;
};


/**
 * optional google.protobuf.Duration start_to_close_timeout = 8;
 * @return {?proto.google.protobuf.Duration}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.getStartToCloseTimeout = function() {
  return /** @type{?proto.google.protobuf.Duration} */ (
    jspb.Message.getWrapperField(this, google_protobuf_duration_pb.Duration, 8));
};


/**
 * @param {?proto.google.protobuf.Duration|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
*/
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.setStartToCloseTimeout = function(value) {
  return jspb.Message.setWrapperField(this, 8, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.clearStartToCloseTimeout = function() {
  return this.setStartToCloseTimeout(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.hasStartToCloseTimeout = function() {
  return jspb.Message.getField(this, 8) != null;
};


/**
 * optional google.protobuf.Duration heartbeat_timeout = 9;
 * @return {?proto.google.protobuf.Duration}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.getHeartbeatTimeout = function() {
  return /** @type{?proto.google.protobuf.Duration} */ (
    jspb.Message.getWrapperField(this, google_protobuf_duration_pb.Duration, 9));
};


/**
 * @param {?proto.google.protobuf.Duration|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
*/
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.setHeartbeatTimeout = function(value) {
  return jspb.Message.setWrapperField(this, 9, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.clearHeartbeatTimeout = function() {
  return this.setHeartbeatTimeout(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.hasHeartbeatTimeout = function() {
  return jspb.Message.getField(this, 9) != null;
};


/**
 * optional temporal.api.common.v1.RetryPolicy retry_policy = 10;
 * @return {?proto.temporal.api.common.v1.RetryPolicy}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.getRetryPolicy = function() {
  return /** @type{?proto.temporal.api.common.v1.RetryPolicy} */ (
    jspb.Message.getWrapperField(this, temporal_api_common_v1_message_pb.RetryPolicy, 10));
};


/**
 * @param {?proto.temporal.api.common.v1.RetryPolicy|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
*/
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.setRetryPolicy = function(value) {
  return jspb.Message.setWrapperField(this, 10, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.clearRetryPolicy = function() {
  return this.setRetryPolicy(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.hasRetryPolicy = function() {
  return jspb.Message.getField(this, 10) != null;
};


/**
 * optional google.protobuf.Empty is_local = 11;
 * @return {?proto.google.protobuf.Empty}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.getIsLocal = function() {
  return /** @type{?proto.google.protobuf.Empty} */ (
    jspb.Message.getWrapperField(this, google_protobuf_empty_pb.Empty, 11));
};


/**
 * @param {?proto.google.protobuf.Empty|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
*/
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.setIsLocal = function(value) {
  return jspb.Message.setOneofWrapperField(this, 11, proto.temporal.omes.kitchen_sink.ExecuteActivityAction.oneofGroups_[1], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.clearIsLocal = function() {
  return this.setIsLocal(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.hasIsLocal = function() {
  return jspb.Message.getField(this, 11) != null;
};


/**
 * optional RemoteActivityOptions remote = 12;
 * @return {?proto.temporal.omes.kitchen_sink.RemoteActivityOptions}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.getRemote = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.RemoteActivityOptions} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.RemoteActivityOptions, 12));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.RemoteActivityOptions|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
*/
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.setRemote = function(value) {
  return jspb.Message.setOneofWrapperField(this, 12, proto.temporal.omes.kitchen_sink.ExecuteActivityAction.oneofGroups_[1], value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.clearRemote = function() {
  return this.setRemote(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.hasRemote = function() {
  return jspb.Message.getField(this, 12) != null;
};


/**
 * optional AwaitableChoice awaitable_choice = 13;
 * @return {?proto.temporal.omes.kitchen_sink.AwaitableChoice}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.getAwaitableChoice = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.AwaitableChoice} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.AwaitableChoice, 13));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.AwaitableChoice|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
*/
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.setAwaitableChoice = function(value) {
  return jspb.Message.setWrapperField(this, 13, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.clearAwaitableChoice = function() {
  return this.setAwaitableChoice(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.hasAwaitableChoice = function() {
  return jspb.Message.getField(this, 13) != null;
};


/**
 * optional temporal.api.common.v1.Priority priority = 15;
 * @return {?proto.temporal.api.common.v1.Priority}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.getPriority = function() {
  return /** @type{?proto.temporal.api.common.v1.Priority} */ (
    jspb.Message.getWrapperField(this, temporal_api_common_v1_message_pb.Priority, 15));
};


/**
 * @param {?proto.temporal.api.common.v1.Priority|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
*/
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.setPriority = function(value) {
  return jspb.Message.setWrapperField(this, 15, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.clearPriority = function() {
  return this.setPriority(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.hasPriority = function() {
  return jspb.Message.getField(this, 15) != null;
};


/**
 * optional string fairness_key = 16;
 * @return {string}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.getFairnessKey = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 16, ""));
};


/**
 * @param {string} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.setFairnessKey = function(value) {
  return jspb.Message.setProto3StringField(this, 16, value);
};


/**
 * optional float fairness_weight = 17;
 * @return {number}
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.getFairnessWeight = function() {
  return /** @type {number} */ (jspb.Message.getFloatingPointFieldWithDefault(this, 17, 0.0));
};


/**
 * @param {number} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteActivityAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteActivityAction.prototype.setFairnessWeight = function(value) {
  return jspb.Message.setProto3FloatField(this, 17, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.repeatedFields_ = [6];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.toObject = function(includeInstance, msg) {
  var f, obj = {
namespace: jspb.Message.getFieldWithDefault(msg, 2, ""),
workflowId: jspb.Message.getFieldWithDefault(msg, 3, ""),
workflowType: jspb.Message.getFieldWithDefault(msg, 4, ""),
taskQueue: jspb.Message.getFieldWithDefault(msg, 5, ""),
inputList: jspb.Message.toObjectList(msg.getInputList(),
    temporal_api_common_v1_message_pb.Payload.toObject, includeInstance),
workflowExecutionTimeout: (f = msg.getWorkflowExecutionTimeout()) && google_protobuf_duration_pb.Duration.toObject(includeInstance, f),
workflowRunTimeout: (f = msg.getWorkflowRunTimeout()) && google_protobuf_duration_pb.Duration.toObject(includeInstance, f),
workflowTaskTimeout: (f = msg.getWorkflowTaskTimeout()) && google_protobuf_duration_pb.Duration.toObject(includeInstance, f),
parentClosePolicy: jspb.Message.getFieldWithDefault(msg, 10, 0),
workflowIdReusePolicy: jspb.Message.getFieldWithDefault(msg, 12, 0),
retryPolicy: (f = msg.getRetryPolicy()) && temporal_api_common_v1_message_pb.RetryPolicy.toObject(includeInstance, f),
cronSchedule: jspb.Message.getFieldWithDefault(msg, 14, ""),
headersMap: (f = msg.getHeadersMap()) ? f.toObject(includeInstance, proto.temporal.api.common.v1.Payload.toObject) : [],
memoMap: (f = msg.getMemoMap()) ? f.toObject(includeInstance, proto.temporal.api.common.v1.Payload.toObject) : [],
searchAttributesMap: (f = msg.getSearchAttributesMap()) ? f.toObject(includeInstance, proto.temporal.api.common.v1.Payload.toObject) : [],
cancellationType: jspb.Message.getFieldWithDefault(msg, 18, 0),
versioningIntent: jspb.Message.getFieldWithDefault(msg, 19, 0),
awaitableChoice: (f = msg.getAwaitableChoice()) && proto.temporal.omes.kitchen_sink.AwaitableChoice.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction}
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction;
  return proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction}
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setNamespace(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setWorkflowId(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setWorkflowType(value);
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.setTaskQueue(value);
      break;
    case 6:
      var value = new temporal_api_common_v1_message_pb.Payload;
      reader.readMessage(value,temporal_api_common_v1_message_pb.Payload.deserializeBinaryFromReader);
      msg.addInput(value);
      break;
    case 7:
      var value = new google_protobuf_duration_pb.Duration;
      reader.readMessage(value,google_protobuf_duration_pb.Duration.deserializeBinaryFromReader);
      msg.setWorkflowExecutionTimeout(value);
      break;
    case 8:
      var value = new google_protobuf_duration_pb.Duration;
      reader.readMessage(value,google_protobuf_duration_pb.Duration.deserializeBinaryFromReader);
      msg.setWorkflowRunTimeout(value);
      break;
    case 9:
      var value = new google_protobuf_duration_pb.Duration;
      reader.readMessage(value,google_protobuf_duration_pb.Duration.deserializeBinaryFromReader);
      msg.setWorkflowTaskTimeout(value);
      break;
    case 10:
      var value = /** @type {!proto.temporal.omes.kitchen_sink.ParentClosePolicy} */ (reader.readEnum());
      msg.setParentClosePolicy(value);
      break;
    case 12:
      var value = /** @type {!proto.temporal.api.enums.v1.WorkflowIdReusePolicy} */ (reader.readEnum());
      msg.setWorkflowIdReusePolicy(value);
      break;
    case 13:
      var value = new temporal_api_common_v1_message_pb.RetryPolicy;
      reader.readMessage(value,temporal_api_common_v1_message_pb.RetryPolicy.deserializeBinaryFromReader);
      msg.setRetryPolicy(value);
      break;
    case 14:
      var value = /** @type {string} */ (reader.readString());
      msg.setCronSchedule(value);
      break;
    case 15:
      var value = msg.getHeadersMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.temporal.api.common.v1.Payload.deserializeBinaryFromReader, "", new proto.temporal.api.common.v1.Payload());
         });
      break;
    case 16:
      var value = msg.getMemoMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.temporal.api.common.v1.Payload.deserializeBinaryFromReader, "", new proto.temporal.api.common.v1.Payload());
         });
      break;
    case 17:
      var value = msg.getSearchAttributesMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.temporal.api.common.v1.Payload.deserializeBinaryFromReader, "", new proto.temporal.api.common.v1.Payload());
         });
      break;
    case 18:
      var value = /** @type {!proto.temporal.omes.kitchen_sink.ChildWorkflowCancellationType} */ (reader.readEnum());
      msg.setCancellationType(value);
      break;
    case 19:
      var value = /** @type {!proto.temporal.omes.kitchen_sink.VersioningIntent} */ (reader.readEnum());
      msg.setVersioningIntent(value);
      break;
    case 20:
      var value = new proto.temporal.omes.kitchen_sink.AwaitableChoice;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.AwaitableChoice.deserializeBinaryFromReader);
      msg.setAwaitableChoice(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getNamespace();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getWorkflowId();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getWorkflowType();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
  f = message.getTaskQueue();
  if (f.length > 0) {
    writer.writeString(
      5,
      f
    );
  }
  f = message.getInputList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      6,
      f,
      temporal_api_common_v1_message_pb.Payload.serializeBinaryToWriter
    );
  }
  f = message.getWorkflowExecutionTimeout();
  if (f != null) {
    writer.writeMessage(
      7,
      f,
      google_protobuf_duration_pb.Duration.serializeBinaryToWriter
    );
  }
  f = message.getWorkflowRunTimeout();
  if (f != null) {
    writer.writeMessage(
      8,
      f,
      google_protobuf_duration_pb.Duration.serializeBinaryToWriter
    );
  }
  f = message.getWorkflowTaskTimeout();
  if (f != null) {
    writer.writeMessage(
      9,
      f,
      google_protobuf_duration_pb.Duration.serializeBinaryToWriter
    );
  }
  f = message.getParentClosePolicy();
  if (f !== 0.0) {
    writer.writeEnum(
      10,
      f
    );
  }
  f = message.getWorkflowIdReusePolicy();
  if (f !== 0.0) {
    writer.writeEnum(
      12,
      f
    );
  }
  f = message.getRetryPolicy();
  if (f != null) {
    writer.writeMessage(
      13,
      f,
      temporal_api_common_v1_message_pb.RetryPolicy.serializeBinaryToWriter
    );
  }
  f = message.getCronSchedule();
  if (f.length > 0) {
    writer.writeString(
      14,
      f
    );
  }
  f = message.getHeadersMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(15, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.temporal.api.common.v1.Payload.serializeBinaryToWriter);
  }
  f = message.getMemoMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(16, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.temporal.api.common.v1.Payload.serializeBinaryToWriter);
  }
  f = message.getSearchAttributesMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(17, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.temporal.api.common.v1.Payload.serializeBinaryToWriter);
  }
  f = message.getCancellationType();
  if (f !== 0.0) {
    writer.writeEnum(
      18,
      f
    );
  }
  f = message.getVersioningIntent();
  if (f !== 0.0) {
    writer.writeEnum(
      19,
      f
    );
  }
  f = message.getAwaitableChoice();
  if (f != null) {
    writer.writeMessage(
      20,
      f,
      proto.temporal.omes.kitchen_sink.AwaitableChoice.serializeBinaryToWriter
    );
  }
};


/**
 * optional string namespace = 2;
 * @return {string}
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.getNamespace = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.setNamespace = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string workflow_id = 3;
 * @return {string}
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.getWorkflowId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.setWorkflowId = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional string workflow_type = 4;
 * @return {string}
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.getWorkflowType = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.setWorkflowType = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * optional string task_queue = 5;
 * @return {string}
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.getTaskQueue = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 5, ""));
};


/**
 * @param {string} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.setTaskQueue = function(value) {
  return jspb.Message.setProto3StringField(this, 5, value);
};


/**
 * repeated temporal.api.common.v1.Payload input = 6;
 * @return {!Array<!proto.temporal.api.common.v1.Payload>}
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.getInputList = function() {
  return /** @type{!Array<!proto.temporal.api.common.v1.Payload>} */ (
    jspb.Message.getRepeatedWrapperField(this, temporal_api_common_v1_message_pb.Payload, 6));
};


/**
 * @param {!Array<!proto.temporal.api.common.v1.Payload>} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction} returns this
*/
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.setInputList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 6, value);
};


/**
 * @param {!proto.temporal.api.common.v1.Payload=} opt_value
 * @param {number=} opt_index
 * @return {!proto.temporal.api.common.v1.Payload}
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.addInput = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 6, opt_value, proto.temporal.api.common.v1.Payload, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.clearInputList = function() {
  return this.setInputList([]);
};


/**
 * optional google.protobuf.Duration workflow_execution_timeout = 7;
 * @return {?proto.google.protobuf.Duration}
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.getWorkflowExecutionTimeout = function() {
  return /** @type{?proto.google.protobuf.Duration} */ (
    jspb.Message.getWrapperField(this, google_protobuf_duration_pb.Duration, 7));
};


/**
 * @param {?proto.google.protobuf.Duration|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction} returns this
*/
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.setWorkflowExecutionTimeout = function(value) {
  return jspb.Message.setWrapperField(this, 7, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.clearWorkflowExecutionTimeout = function() {
  return this.setWorkflowExecutionTimeout(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.hasWorkflowExecutionTimeout = function() {
  return jspb.Message.getField(this, 7) != null;
};


/**
 * optional google.protobuf.Duration workflow_run_timeout = 8;
 * @return {?proto.google.protobuf.Duration}
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.getWorkflowRunTimeout = function() {
  return /** @type{?proto.google.protobuf.Duration} */ (
    jspb.Message.getWrapperField(this, google_protobuf_duration_pb.Duration, 8));
};


/**
 * @param {?proto.google.protobuf.Duration|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction} returns this
*/
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.setWorkflowRunTimeout = function(value) {
  return jspb.Message.setWrapperField(this, 8, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.clearWorkflowRunTimeout = function() {
  return this.setWorkflowRunTimeout(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.hasWorkflowRunTimeout = function() {
  return jspb.Message.getField(this, 8) != null;
};


/**
 * optional google.protobuf.Duration workflow_task_timeout = 9;
 * @return {?proto.google.protobuf.Duration}
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.getWorkflowTaskTimeout = function() {
  return /** @type{?proto.google.protobuf.Duration} */ (
    jspb.Message.getWrapperField(this, google_protobuf_duration_pb.Duration, 9));
};


/**
 * @param {?proto.google.protobuf.Duration|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction} returns this
*/
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.setWorkflowTaskTimeout = function(value) {
  return jspb.Message.setWrapperField(this, 9, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.clearWorkflowTaskTimeout = function() {
  return this.setWorkflowTaskTimeout(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.hasWorkflowTaskTimeout = function() {
  return jspb.Message.getField(this, 9) != null;
};


/**
 * optional ParentClosePolicy parent_close_policy = 10;
 * @return {!proto.temporal.omes.kitchen_sink.ParentClosePolicy}
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.getParentClosePolicy = function() {
  return /** @type {!proto.temporal.omes.kitchen_sink.ParentClosePolicy} */ (jspb.Message.getFieldWithDefault(this, 10, 0));
};


/**
 * @param {!proto.temporal.omes.kitchen_sink.ParentClosePolicy} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.setParentClosePolicy = function(value) {
  return jspb.Message.setProto3EnumField(this, 10, value);
};


/**
 * optional temporal.api.enums.v1.WorkflowIdReusePolicy workflow_id_reuse_policy = 12;
 * @return {!proto.temporal.api.enums.v1.WorkflowIdReusePolicy}
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.getWorkflowIdReusePolicy = function() {
  return /** @type {!proto.temporal.api.enums.v1.WorkflowIdReusePolicy} */ (jspb.Message.getFieldWithDefault(this, 12, 0));
};


/**
 * @param {!proto.temporal.api.enums.v1.WorkflowIdReusePolicy} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.setWorkflowIdReusePolicy = function(value) {
  return jspb.Message.setProto3EnumField(this, 12, value);
};


/**
 * optional temporal.api.common.v1.RetryPolicy retry_policy = 13;
 * @return {?proto.temporal.api.common.v1.RetryPolicy}
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.getRetryPolicy = function() {
  return /** @type{?proto.temporal.api.common.v1.RetryPolicy} */ (
    jspb.Message.getWrapperField(this, temporal_api_common_v1_message_pb.RetryPolicy, 13));
};


/**
 * @param {?proto.temporal.api.common.v1.RetryPolicy|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction} returns this
*/
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.setRetryPolicy = function(value) {
  return jspb.Message.setWrapperField(this, 13, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.clearRetryPolicy = function() {
  return this.setRetryPolicy(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.hasRetryPolicy = function() {
  return jspb.Message.getField(this, 13) != null;
};


/**
 * optional string cron_schedule = 14;
 * @return {string}
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.getCronSchedule = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 14, ""));
};


/**
 * @param {string} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.setCronSchedule = function(value) {
  return jspb.Message.setProto3StringField(this, 14, value);
};


/**
 * map<string, temporal.api.common.v1.Payload> headers = 15;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.temporal.api.common.v1.Payload>}
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.getHeadersMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.temporal.api.common.v1.Payload>} */ (
      jspb.Message.getMapField(this, 15, opt_noLazyCreate,
      proto.temporal.api.common.v1.Payload));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.clearHeadersMap = function() {
  this.getHeadersMap().clear();
  return this;
};


/**
 * map<string, temporal.api.common.v1.Payload> memo = 16;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.temporal.api.common.v1.Payload>}
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.getMemoMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.temporal.api.common.v1.Payload>} */ (
      jspb.Message.getMapField(this, 16, opt_noLazyCreate,
      proto.temporal.api.common.v1.Payload));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.clearMemoMap = function() {
  this.getMemoMap().clear();
  return this;
};


/**
 * map<string, temporal.api.common.v1.Payload> search_attributes = 17;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.temporal.api.common.v1.Payload>}
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.getSearchAttributesMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.temporal.api.common.v1.Payload>} */ (
      jspb.Message.getMapField(this, 17, opt_noLazyCreate,
      proto.temporal.api.common.v1.Payload));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.clearSearchAttributesMap = function() {
  this.getSearchAttributesMap().clear();
  return this;
};


/**
 * optional ChildWorkflowCancellationType cancellation_type = 18;
 * @return {!proto.temporal.omes.kitchen_sink.ChildWorkflowCancellationType}
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.getCancellationType = function() {
  return /** @type {!proto.temporal.omes.kitchen_sink.ChildWorkflowCancellationType} */ (jspb.Message.getFieldWithDefault(this, 18, 0));
};


/**
 * @param {!proto.temporal.omes.kitchen_sink.ChildWorkflowCancellationType} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.setCancellationType = function(value) {
  return jspb.Message.setProto3EnumField(this, 18, value);
};


/**
 * optional VersioningIntent versioning_intent = 19;
 * @return {!proto.temporal.omes.kitchen_sink.VersioningIntent}
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.getVersioningIntent = function() {
  return /** @type {!proto.temporal.omes.kitchen_sink.VersioningIntent} */ (jspb.Message.getFieldWithDefault(this, 19, 0));
};


/**
 * @param {!proto.temporal.omes.kitchen_sink.VersioningIntent} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.setVersioningIntent = function(value) {
  return jspb.Message.setProto3EnumField(this, 19, value);
};


/**
 * optional AwaitableChoice awaitable_choice = 20;
 * @return {?proto.temporal.omes.kitchen_sink.AwaitableChoice}
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.getAwaitableChoice = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.AwaitableChoice} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.AwaitableChoice, 20));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.AwaitableChoice|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction} returns this
*/
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.setAwaitableChoice = function(value) {
  return jspb.Message.setWrapperField(this, 20, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.clearAwaitableChoice = function() {
  return this.setAwaitableChoice(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.prototype.hasAwaitableChoice = function() {
  return jspb.Message.getField(this, 20) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.AwaitWorkflowState.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.AwaitWorkflowState.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.AwaitWorkflowState} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.AwaitWorkflowState.toObject = function(includeInstance, msg) {
  var f, obj = {
key: jspb.Message.getFieldWithDefault(msg, 1, ""),
value: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.AwaitWorkflowState}
 */
proto.temporal.omes.kitchen_sink.AwaitWorkflowState.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.AwaitWorkflowState;
  return proto.temporal.omes.kitchen_sink.AwaitWorkflowState.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.AwaitWorkflowState} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.AwaitWorkflowState}
 */
proto.temporal.omes.kitchen_sink.AwaitWorkflowState.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setKey(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setValue(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.AwaitWorkflowState.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.AwaitWorkflowState.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.AwaitWorkflowState} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.AwaitWorkflowState.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getKey();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getValue();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional string key = 1;
 * @return {string}
 */
proto.temporal.omes.kitchen_sink.AwaitWorkflowState.prototype.getKey = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.temporal.omes.kitchen_sink.AwaitWorkflowState} returns this
 */
proto.temporal.omes.kitchen_sink.AwaitWorkflowState.prototype.setKey = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string value = 2;
 * @return {string}
 */
proto.temporal.omes.kitchen_sink.AwaitWorkflowState.prototype.getValue = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.temporal.omes.kitchen_sink.AwaitWorkflowState} returns this
 */
proto.temporal.omes.kitchen_sink.AwaitWorkflowState.prototype.setValue = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.temporal.omes.kitchen_sink.SendSignalAction.repeatedFields_ = [4];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.SendSignalAction.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.SendSignalAction.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.SendSignalAction} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.SendSignalAction.toObject = function(includeInstance, msg) {
  var f, obj = {
workflowId: jspb.Message.getFieldWithDefault(msg, 1, ""),
runId: jspb.Message.getFieldWithDefault(msg, 2, ""),
signalName: jspb.Message.getFieldWithDefault(msg, 3, ""),
argsList: jspb.Message.toObjectList(msg.getArgsList(),
    temporal_api_common_v1_message_pb.Payload.toObject, includeInstance),
headersMap: (f = msg.getHeadersMap()) ? f.toObject(includeInstance, proto.temporal.api.common.v1.Payload.toObject) : [],
awaitableChoice: (f = msg.getAwaitableChoice()) && proto.temporal.omes.kitchen_sink.AwaitableChoice.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.SendSignalAction}
 */
proto.temporal.omes.kitchen_sink.SendSignalAction.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.SendSignalAction;
  return proto.temporal.omes.kitchen_sink.SendSignalAction.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.SendSignalAction} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.SendSignalAction}
 */
proto.temporal.omes.kitchen_sink.SendSignalAction.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setWorkflowId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setRunId(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setSignalName(value);
      break;
    case 4:
      var value = new temporal_api_common_v1_message_pb.Payload;
      reader.readMessage(value,temporal_api_common_v1_message_pb.Payload.deserializeBinaryFromReader);
      msg.addArgs(value);
      break;
    case 5:
      var value = msg.getHeadersMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.temporal.api.common.v1.Payload.deserializeBinaryFromReader, "", new proto.temporal.api.common.v1.Payload());
         });
      break;
    case 6:
      var value = new proto.temporal.omes.kitchen_sink.AwaitableChoice;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.AwaitableChoice.deserializeBinaryFromReader);
      msg.setAwaitableChoice(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.SendSignalAction.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.SendSignalAction.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.SendSignalAction} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.SendSignalAction.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getWorkflowId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getRunId();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getSignalName();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getArgsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      4,
      f,
      temporal_api_common_v1_message_pb.Payload.serializeBinaryToWriter
    );
  }
  f = message.getHeadersMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(5, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.temporal.api.common.v1.Payload.serializeBinaryToWriter);
  }
  f = message.getAwaitableChoice();
  if (f != null) {
    writer.writeMessage(
      6,
      f,
      proto.temporal.omes.kitchen_sink.AwaitableChoice.serializeBinaryToWriter
    );
  }
};


/**
 * optional string workflow_id = 1;
 * @return {string}
 */
proto.temporal.omes.kitchen_sink.SendSignalAction.prototype.getWorkflowId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.temporal.omes.kitchen_sink.SendSignalAction} returns this
 */
proto.temporal.omes.kitchen_sink.SendSignalAction.prototype.setWorkflowId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string run_id = 2;
 * @return {string}
 */
proto.temporal.omes.kitchen_sink.SendSignalAction.prototype.getRunId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.temporal.omes.kitchen_sink.SendSignalAction} returns this
 */
proto.temporal.omes.kitchen_sink.SendSignalAction.prototype.setRunId = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string signal_name = 3;
 * @return {string}
 */
proto.temporal.omes.kitchen_sink.SendSignalAction.prototype.getSignalName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.temporal.omes.kitchen_sink.SendSignalAction} returns this
 */
proto.temporal.omes.kitchen_sink.SendSignalAction.prototype.setSignalName = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * repeated temporal.api.common.v1.Payload args = 4;
 * @return {!Array<!proto.temporal.api.common.v1.Payload>}
 */
proto.temporal.omes.kitchen_sink.SendSignalAction.prototype.getArgsList = function() {
  return /** @type{!Array<!proto.temporal.api.common.v1.Payload>} */ (
    jspb.Message.getRepeatedWrapperField(this, temporal_api_common_v1_message_pb.Payload, 4));
};


/**
 * @param {!Array<!proto.temporal.api.common.v1.Payload>} value
 * @return {!proto.temporal.omes.kitchen_sink.SendSignalAction} returns this
*/
proto.temporal.omes.kitchen_sink.SendSignalAction.prototype.setArgsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 4, value);
};


/**
 * @param {!proto.temporal.api.common.v1.Payload=} opt_value
 * @param {number=} opt_index
 * @return {!proto.temporal.api.common.v1.Payload}
 */
proto.temporal.omes.kitchen_sink.SendSignalAction.prototype.addArgs = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 4, opt_value, proto.temporal.api.common.v1.Payload, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.temporal.omes.kitchen_sink.SendSignalAction} returns this
 */
proto.temporal.omes.kitchen_sink.SendSignalAction.prototype.clearArgsList = function() {
  return this.setArgsList([]);
};


/**
 * map<string, temporal.api.common.v1.Payload> headers = 5;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.temporal.api.common.v1.Payload>}
 */
proto.temporal.omes.kitchen_sink.SendSignalAction.prototype.getHeadersMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.temporal.api.common.v1.Payload>} */ (
      jspb.Message.getMapField(this, 5, opt_noLazyCreate,
      proto.temporal.api.common.v1.Payload));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.temporal.omes.kitchen_sink.SendSignalAction} returns this
 */
proto.temporal.omes.kitchen_sink.SendSignalAction.prototype.clearHeadersMap = function() {
  this.getHeadersMap().clear();
  return this;
};


/**
 * optional AwaitableChoice awaitable_choice = 6;
 * @return {?proto.temporal.omes.kitchen_sink.AwaitableChoice}
 */
proto.temporal.omes.kitchen_sink.SendSignalAction.prototype.getAwaitableChoice = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.AwaitableChoice} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.AwaitableChoice, 6));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.AwaitableChoice|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.SendSignalAction} returns this
*/
proto.temporal.omes.kitchen_sink.SendSignalAction.prototype.setAwaitableChoice = function(value) {
  return jspb.Message.setWrapperField(this, 6, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.SendSignalAction} returns this
 */
proto.temporal.omes.kitchen_sink.SendSignalAction.prototype.clearAwaitableChoice = function() {
  return this.setAwaitableChoice(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.SendSignalAction.prototype.hasAwaitableChoice = function() {
  return jspb.Message.getField(this, 6) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.CancelWorkflowAction.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.CancelWorkflowAction.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.CancelWorkflowAction} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.CancelWorkflowAction.toObject = function(includeInstance, msg) {
  var f, obj = {
workflowId: jspb.Message.getFieldWithDefault(msg, 1, ""),
runId: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.CancelWorkflowAction}
 */
proto.temporal.omes.kitchen_sink.CancelWorkflowAction.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.CancelWorkflowAction;
  return proto.temporal.omes.kitchen_sink.CancelWorkflowAction.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.CancelWorkflowAction} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.CancelWorkflowAction}
 */
proto.temporal.omes.kitchen_sink.CancelWorkflowAction.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setWorkflowId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setRunId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.CancelWorkflowAction.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.CancelWorkflowAction.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.CancelWorkflowAction} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.CancelWorkflowAction.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getWorkflowId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getRunId();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional string workflow_id = 1;
 * @return {string}
 */
proto.temporal.omes.kitchen_sink.CancelWorkflowAction.prototype.getWorkflowId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.temporal.omes.kitchen_sink.CancelWorkflowAction} returns this
 */
proto.temporal.omes.kitchen_sink.CancelWorkflowAction.prototype.setWorkflowId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string run_id = 2;
 * @return {string}
 */
proto.temporal.omes.kitchen_sink.CancelWorkflowAction.prototype.getRunId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.temporal.omes.kitchen_sink.CancelWorkflowAction} returns this
 */
proto.temporal.omes.kitchen_sink.CancelWorkflowAction.prototype.setRunId = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.SetPatchMarkerAction.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.SetPatchMarkerAction.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.SetPatchMarkerAction} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.SetPatchMarkerAction.toObject = function(includeInstance, msg) {
  var f, obj = {
patchId: jspb.Message.getFieldWithDefault(msg, 1, ""),
deprecated: jspb.Message.getBooleanFieldWithDefault(msg, 2, false),
innerAction: (f = msg.getInnerAction()) && proto.temporal.omes.kitchen_sink.Action.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.SetPatchMarkerAction}
 */
proto.temporal.omes.kitchen_sink.SetPatchMarkerAction.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.SetPatchMarkerAction;
  return proto.temporal.omes.kitchen_sink.SetPatchMarkerAction.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.SetPatchMarkerAction} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.SetPatchMarkerAction}
 */
proto.temporal.omes.kitchen_sink.SetPatchMarkerAction.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setPatchId(value);
      break;
    case 2:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setDeprecated(value);
      break;
    case 3:
      var value = new proto.temporal.omes.kitchen_sink.Action;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.Action.deserializeBinaryFromReader);
      msg.setInnerAction(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.SetPatchMarkerAction.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.SetPatchMarkerAction.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.SetPatchMarkerAction} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.SetPatchMarkerAction.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getPatchId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getDeprecated();
  if (f) {
    writer.writeBool(
      2,
      f
    );
  }
  f = message.getInnerAction();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      proto.temporal.omes.kitchen_sink.Action.serializeBinaryToWriter
    );
  }
};


/**
 * optional string patch_id = 1;
 * @return {string}
 */
proto.temporal.omes.kitchen_sink.SetPatchMarkerAction.prototype.getPatchId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.temporal.omes.kitchen_sink.SetPatchMarkerAction} returns this
 */
proto.temporal.omes.kitchen_sink.SetPatchMarkerAction.prototype.setPatchId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional bool deprecated = 2;
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.SetPatchMarkerAction.prototype.getDeprecated = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 2, false));
};


/**
 * @param {boolean} value
 * @return {!proto.temporal.omes.kitchen_sink.SetPatchMarkerAction} returns this
 */
proto.temporal.omes.kitchen_sink.SetPatchMarkerAction.prototype.setDeprecated = function(value) {
  return jspb.Message.setProto3BooleanField(this, 2, value);
};


/**
 * optional Action inner_action = 3;
 * @return {?proto.temporal.omes.kitchen_sink.Action}
 */
proto.temporal.omes.kitchen_sink.SetPatchMarkerAction.prototype.getInnerAction = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.Action} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.Action, 3));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.Action|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.SetPatchMarkerAction} returns this
*/
proto.temporal.omes.kitchen_sink.SetPatchMarkerAction.prototype.setInnerAction = function(value) {
  return jspb.Message.setWrapperField(this, 3, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.SetPatchMarkerAction} returns this
 */
proto.temporal.omes.kitchen_sink.SetPatchMarkerAction.prototype.clearInnerAction = function() {
  return this.setInnerAction(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.SetPatchMarkerAction.prototype.hasInnerAction = function() {
  return jspb.Message.getField(this, 3) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction.toObject = function(includeInstance, msg) {
  var f, obj = {
searchAttributesMap: (f = msg.getSearchAttributesMap()) ? f.toObject(includeInstance, proto.temporal.api.common.v1.Payload.toObject) : []
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction}
 */
proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction;
  return proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction}
 */
proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = msg.getSearchAttributesMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.temporal.api.common.v1.Payload.deserializeBinaryFromReader, "", new proto.temporal.api.common.v1.Payload());
         });
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getSearchAttributesMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(1, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.temporal.api.common.v1.Payload.serializeBinaryToWriter);
  }
};


/**
 * map<string, temporal.api.common.v1.Payload> search_attributes = 1;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.temporal.api.common.v1.Payload>}
 */
proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction.prototype.getSearchAttributesMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.temporal.api.common.v1.Payload>} */ (
      jspb.Message.getMapField(this, 1, opt_noLazyCreate,
      proto.temporal.api.common.v1.Payload));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction} returns this
 */
proto.temporal.omes.kitchen_sink.UpsertSearchAttributesAction.prototype.clearSearchAttributesMap = function() {
  this.getSearchAttributesMap().clear();
  return this;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.UpsertMemoAction.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.UpsertMemoAction.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.UpsertMemoAction} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.UpsertMemoAction.toObject = function(includeInstance, msg) {
  var f, obj = {
upsertedMemo: (f = msg.getUpsertedMemo()) && temporal_api_common_v1_message_pb.Memo.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.UpsertMemoAction}
 */
proto.temporal.omes.kitchen_sink.UpsertMemoAction.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.UpsertMemoAction;
  return proto.temporal.omes.kitchen_sink.UpsertMemoAction.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.UpsertMemoAction} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.UpsertMemoAction}
 */
proto.temporal.omes.kitchen_sink.UpsertMemoAction.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new temporal_api_common_v1_message_pb.Memo;
      reader.readMessage(value,temporal_api_common_v1_message_pb.Memo.deserializeBinaryFromReader);
      msg.setUpsertedMemo(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.UpsertMemoAction.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.UpsertMemoAction.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.UpsertMemoAction} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.UpsertMemoAction.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getUpsertedMemo();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      temporal_api_common_v1_message_pb.Memo.serializeBinaryToWriter
    );
  }
};


/**
 * optional temporal.api.common.v1.Memo upserted_memo = 1;
 * @return {?proto.temporal.api.common.v1.Memo}
 */
proto.temporal.omes.kitchen_sink.UpsertMemoAction.prototype.getUpsertedMemo = function() {
  return /** @type{?proto.temporal.api.common.v1.Memo} */ (
    jspb.Message.getWrapperField(this, temporal_api_common_v1_message_pb.Memo, 1));
};


/**
 * @param {?proto.temporal.api.common.v1.Memo|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.UpsertMemoAction} returns this
*/
proto.temporal.omes.kitchen_sink.UpsertMemoAction.prototype.setUpsertedMemo = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.UpsertMemoAction} returns this
 */
proto.temporal.omes.kitchen_sink.UpsertMemoAction.prototype.clearUpsertedMemo = function() {
  return this.setUpsertedMemo(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.UpsertMemoAction.prototype.hasUpsertedMemo = function() {
  return jspb.Message.getField(this, 1) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.ReturnResultAction.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.ReturnResultAction.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.ReturnResultAction} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.ReturnResultAction.toObject = function(includeInstance, msg) {
  var f, obj = {
returnThis: (f = msg.getReturnThis()) && temporal_api_common_v1_message_pb.Payload.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.ReturnResultAction}
 */
proto.temporal.omes.kitchen_sink.ReturnResultAction.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.ReturnResultAction;
  return proto.temporal.omes.kitchen_sink.ReturnResultAction.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.ReturnResultAction} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.ReturnResultAction}
 */
proto.temporal.omes.kitchen_sink.ReturnResultAction.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new temporal_api_common_v1_message_pb.Payload;
      reader.readMessage(value,temporal_api_common_v1_message_pb.Payload.deserializeBinaryFromReader);
      msg.setReturnThis(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.ReturnResultAction.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.ReturnResultAction.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.ReturnResultAction} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.ReturnResultAction.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getReturnThis();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      temporal_api_common_v1_message_pb.Payload.serializeBinaryToWriter
    );
  }
};


/**
 * optional temporal.api.common.v1.Payload return_this = 1;
 * @return {?proto.temporal.api.common.v1.Payload}
 */
proto.temporal.omes.kitchen_sink.ReturnResultAction.prototype.getReturnThis = function() {
  return /** @type{?proto.temporal.api.common.v1.Payload} */ (
    jspb.Message.getWrapperField(this, temporal_api_common_v1_message_pb.Payload, 1));
};


/**
 * @param {?proto.temporal.api.common.v1.Payload|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ReturnResultAction} returns this
*/
proto.temporal.omes.kitchen_sink.ReturnResultAction.prototype.setReturnThis = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ReturnResultAction} returns this
 */
proto.temporal.omes.kitchen_sink.ReturnResultAction.prototype.clearReturnThis = function() {
  return this.setReturnThis(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ReturnResultAction.prototype.hasReturnThis = function() {
  return jspb.Message.getField(this, 1) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.ReturnErrorAction.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.ReturnErrorAction.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.ReturnErrorAction} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.ReturnErrorAction.toObject = function(includeInstance, msg) {
  var f, obj = {
failure: (f = msg.getFailure()) && temporal_api_failure_v1_message_pb.Failure.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.ReturnErrorAction}
 */
proto.temporal.omes.kitchen_sink.ReturnErrorAction.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.ReturnErrorAction;
  return proto.temporal.omes.kitchen_sink.ReturnErrorAction.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.ReturnErrorAction} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.ReturnErrorAction}
 */
proto.temporal.omes.kitchen_sink.ReturnErrorAction.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new temporal_api_failure_v1_message_pb.Failure;
      reader.readMessage(value,temporal_api_failure_v1_message_pb.Failure.deserializeBinaryFromReader);
      msg.setFailure(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.ReturnErrorAction.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.ReturnErrorAction.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.ReturnErrorAction} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.ReturnErrorAction.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getFailure();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      temporal_api_failure_v1_message_pb.Failure.serializeBinaryToWriter
    );
  }
};


/**
 * optional temporal.api.failure.v1.Failure failure = 1;
 * @return {?proto.temporal.api.failure.v1.Failure}
 */
proto.temporal.omes.kitchen_sink.ReturnErrorAction.prototype.getFailure = function() {
  return /** @type{?proto.temporal.api.failure.v1.Failure} */ (
    jspb.Message.getWrapperField(this, temporal_api_failure_v1_message_pb.Failure, 1));
};


/**
 * @param {?proto.temporal.api.failure.v1.Failure|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ReturnErrorAction} returns this
*/
proto.temporal.omes.kitchen_sink.ReturnErrorAction.prototype.setFailure = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ReturnErrorAction} returns this
 */
proto.temporal.omes.kitchen_sink.ReturnErrorAction.prototype.clearFailure = function() {
  return this.setFailure(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ReturnErrorAction.prototype.hasFailure = function() {
  return jspb.Message.getField(this, 1) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.repeatedFields_ = [3];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.ContinueAsNewAction.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.ContinueAsNewAction} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.toObject = function(includeInstance, msg) {
  var f, obj = {
workflowType: jspb.Message.getFieldWithDefault(msg, 1, ""),
taskQueue: jspb.Message.getFieldWithDefault(msg, 2, ""),
argumentsList: jspb.Message.toObjectList(msg.getArgumentsList(),
    temporal_api_common_v1_message_pb.Payload.toObject, includeInstance),
workflowRunTimeout: (f = msg.getWorkflowRunTimeout()) && google_protobuf_duration_pb.Duration.toObject(includeInstance, f),
workflowTaskTimeout: (f = msg.getWorkflowTaskTimeout()) && google_protobuf_duration_pb.Duration.toObject(includeInstance, f),
memoMap: (f = msg.getMemoMap()) ? f.toObject(includeInstance, proto.temporal.api.common.v1.Payload.toObject) : [],
headersMap: (f = msg.getHeadersMap()) ? f.toObject(includeInstance, proto.temporal.api.common.v1.Payload.toObject) : [],
searchAttributesMap: (f = msg.getSearchAttributesMap()) ? f.toObject(includeInstance, proto.temporal.api.common.v1.Payload.toObject) : [],
retryPolicy: (f = msg.getRetryPolicy()) && temporal_api_common_v1_message_pb.RetryPolicy.toObject(includeInstance, f),
versioningIntent: jspb.Message.getFieldWithDefault(msg, 10, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.ContinueAsNewAction}
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.ContinueAsNewAction;
  return proto.temporal.omes.kitchen_sink.ContinueAsNewAction.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.ContinueAsNewAction} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.ContinueAsNewAction}
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setWorkflowType(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setTaskQueue(value);
      break;
    case 3:
      var value = new temporal_api_common_v1_message_pb.Payload;
      reader.readMessage(value,temporal_api_common_v1_message_pb.Payload.deserializeBinaryFromReader);
      msg.addArguments(value);
      break;
    case 4:
      var value = new google_protobuf_duration_pb.Duration;
      reader.readMessage(value,google_protobuf_duration_pb.Duration.deserializeBinaryFromReader);
      msg.setWorkflowRunTimeout(value);
      break;
    case 5:
      var value = new google_protobuf_duration_pb.Duration;
      reader.readMessage(value,google_protobuf_duration_pb.Duration.deserializeBinaryFromReader);
      msg.setWorkflowTaskTimeout(value);
      break;
    case 6:
      var value = msg.getMemoMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.temporal.api.common.v1.Payload.deserializeBinaryFromReader, "", new proto.temporal.api.common.v1.Payload());
         });
      break;
    case 7:
      var value = msg.getHeadersMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.temporal.api.common.v1.Payload.deserializeBinaryFromReader, "", new proto.temporal.api.common.v1.Payload());
         });
      break;
    case 8:
      var value = msg.getSearchAttributesMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.temporal.api.common.v1.Payload.deserializeBinaryFromReader, "", new proto.temporal.api.common.v1.Payload());
         });
      break;
    case 9:
      var value = new temporal_api_common_v1_message_pb.RetryPolicy;
      reader.readMessage(value,temporal_api_common_v1_message_pb.RetryPolicy.deserializeBinaryFromReader);
      msg.setRetryPolicy(value);
      break;
    case 10:
      var value = /** @type {!proto.temporal.omes.kitchen_sink.VersioningIntent} */ (reader.readEnum());
      msg.setVersioningIntent(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.ContinueAsNewAction.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.ContinueAsNewAction} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getWorkflowType();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getTaskQueue();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getArgumentsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      3,
      f,
      temporal_api_common_v1_message_pb.Payload.serializeBinaryToWriter
    );
  }
  f = message.getWorkflowRunTimeout();
  if (f != null) {
    writer.writeMessage(
      4,
      f,
      google_protobuf_duration_pb.Duration.serializeBinaryToWriter
    );
  }
  f = message.getWorkflowTaskTimeout();
  if (f != null) {
    writer.writeMessage(
      5,
      f,
      google_protobuf_duration_pb.Duration.serializeBinaryToWriter
    );
  }
  f = message.getMemoMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(6, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.temporal.api.common.v1.Payload.serializeBinaryToWriter);
  }
  f = message.getHeadersMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(7, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.temporal.api.common.v1.Payload.serializeBinaryToWriter);
  }
  f = message.getSearchAttributesMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(8, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.temporal.api.common.v1.Payload.serializeBinaryToWriter);
  }
  f = message.getRetryPolicy();
  if (f != null) {
    writer.writeMessage(
      9,
      f,
      temporal_api_common_v1_message_pb.RetryPolicy.serializeBinaryToWriter
    );
  }
  f = message.getVersioningIntent();
  if (f !== 0.0) {
    writer.writeEnum(
      10,
      f
    );
  }
};


/**
 * optional string workflow_type = 1;
 * @return {string}
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.prototype.getWorkflowType = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.temporal.omes.kitchen_sink.ContinueAsNewAction} returns this
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.prototype.setWorkflowType = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string task_queue = 2;
 * @return {string}
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.prototype.getTaskQueue = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.temporal.omes.kitchen_sink.ContinueAsNewAction} returns this
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.prototype.setTaskQueue = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * repeated temporal.api.common.v1.Payload arguments = 3;
 * @return {!Array<!proto.temporal.api.common.v1.Payload>}
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.prototype.getArgumentsList = function() {
  return /** @type{!Array<!proto.temporal.api.common.v1.Payload>} */ (
    jspb.Message.getRepeatedWrapperField(this, temporal_api_common_v1_message_pb.Payload, 3));
};


/**
 * @param {!Array<!proto.temporal.api.common.v1.Payload>} value
 * @return {!proto.temporal.omes.kitchen_sink.ContinueAsNewAction} returns this
*/
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.prototype.setArgumentsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 3, value);
};


/**
 * @param {!proto.temporal.api.common.v1.Payload=} opt_value
 * @param {number=} opt_index
 * @return {!proto.temporal.api.common.v1.Payload}
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.prototype.addArguments = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 3, opt_value, proto.temporal.api.common.v1.Payload, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.temporal.omes.kitchen_sink.ContinueAsNewAction} returns this
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.prototype.clearArgumentsList = function() {
  return this.setArgumentsList([]);
};


/**
 * optional google.protobuf.Duration workflow_run_timeout = 4;
 * @return {?proto.google.protobuf.Duration}
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.prototype.getWorkflowRunTimeout = function() {
  return /** @type{?proto.google.protobuf.Duration} */ (
    jspb.Message.getWrapperField(this, google_protobuf_duration_pb.Duration, 4));
};


/**
 * @param {?proto.google.protobuf.Duration|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ContinueAsNewAction} returns this
*/
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.prototype.setWorkflowRunTimeout = function(value) {
  return jspb.Message.setWrapperField(this, 4, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ContinueAsNewAction} returns this
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.prototype.clearWorkflowRunTimeout = function() {
  return this.setWorkflowRunTimeout(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.prototype.hasWorkflowRunTimeout = function() {
  return jspb.Message.getField(this, 4) != null;
};


/**
 * optional google.protobuf.Duration workflow_task_timeout = 5;
 * @return {?proto.google.protobuf.Duration}
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.prototype.getWorkflowTaskTimeout = function() {
  return /** @type{?proto.google.protobuf.Duration} */ (
    jspb.Message.getWrapperField(this, google_protobuf_duration_pb.Duration, 5));
};


/**
 * @param {?proto.google.protobuf.Duration|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ContinueAsNewAction} returns this
*/
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.prototype.setWorkflowTaskTimeout = function(value) {
  return jspb.Message.setWrapperField(this, 5, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ContinueAsNewAction} returns this
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.prototype.clearWorkflowTaskTimeout = function() {
  return this.setWorkflowTaskTimeout(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.prototype.hasWorkflowTaskTimeout = function() {
  return jspb.Message.getField(this, 5) != null;
};


/**
 * map<string, temporal.api.common.v1.Payload> memo = 6;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.temporal.api.common.v1.Payload>}
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.prototype.getMemoMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.temporal.api.common.v1.Payload>} */ (
      jspb.Message.getMapField(this, 6, opt_noLazyCreate,
      proto.temporal.api.common.v1.Payload));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.temporal.omes.kitchen_sink.ContinueAsNewAction} returns this
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.prototype.clearMemoMap = function() {
  this.getMemoMap().clear();
  return this;
};


/**
 * map<string, temporal.api.common.v1.Payload> headers = 7;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.temporal.api.common.v1.Payload>}
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.prototype.getHeadersMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.temporal.api.common.v1.Payload>} */ (
      jspb.Message.getMapField(this, 7, opt_noLazyCreate,
      proto.temporal.api.common.v1.Payload));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.temporal.omes.kitchen_sink.ContinueAsNewAction} returns this
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.prototype.clearHeadersMap = function() {
  this.getHeadersMap().clear();
  return this;
};


/**
 * map<string, temporal.api.common.v1.Payload> search_attributes = 8;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.temporal.api.common.v1.Payload>}
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.prototype.getSearchAttributesMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.temporal.api.common.v1.Payload>} */ (
      jspb.Message.getMapField(this, 8, opt_noLazyCreate,
      proto.temporal.api.common.v1.Payload));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.temporal.omes.kitchen_sink.ContinueAsNewAction} returns this
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.prototype.clearSearchAttributesMap = function() {
  this.getSearchAttributesMap().clear();
  return this;
};


/**
 * optional temporal.api.common.v1.RetryPolicy retry_policy = 9;
 * @return {?proto.temporal.api.common.v1.RetryPolicy}
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.prototype.getRetryPolicy = function() {
  return /** @type{?proto.temporal.api.common.v1.RetryPolicy} */ (
    jspb.Message.getWrapperField(this, temporal_api_common_v1_message_pb.RetryPolicy, 9));
};


/**
 * @param {?proto.temporal.api.common.v1.RetryPolicy|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ContinueAsNewAction} returns this
*/
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.prototype.setRetryPolicy = function(value) {
  return jspb.Message.setWrapperField(this, 9, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ContinueAsNewAction} returns this
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.prototype.clearRetryPolicy = function() {
  return this.setRetryPolicy(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.prototype.hasRetryPolicy = function() {
  return jspb.Message.getField(this, 9) != null;
};


/**
 * optional VersioningIntent versioning_intent = 10;
 * @return {!proto.temporal.omes.kitchen_sink.VersioningIntent}
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.prototype.getVersioningIntent = function() {
  return /** @type {!proto.temporal.omes.kitchen_sink.VersioningIntent} */ (jspb.Message.getFieldWithDefault(this, 10, 0));
};


/**
 * @param {!proto.temporal.omes.kitchen_sink.VersioningIntent} value
 * @return {!proto.temporal.omes.kitchen_sink.ContinueAsNewAction} returns this
 */
proto.temporal.omes.kitchen_sink.ContinueAsNewAction.prototype.setVersioningIntent = function(value) {
  return jspb.Message.setProto3EnumField(this, 10, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.RemoteActivityOptions.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.RemoteActivityOptions.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.RemoteActivityOptions} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.RemoteActivityOptions.toObject = function(includeInstance, msg) {
  var f, obj = {
cancellationType: jspb.Message.getFieldWithDefault(msg, 1, 0),
doNotEagerlyExecute: jspb.Message.getBooleanFieldWithDefault(msg, 2, false),
versioningIntent: jspb.Message.getFieldWithDefault(msg, 3, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.RemoteActivityOptions}
 */
proto.temporal.omes.kitchen_sink.RemoteActivityOptions.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.RemoteActivityOptions;
  return proto.temporal.omes.kitchen_sink.RemoteActivityOptions.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.RemoteActivityOptions} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.RemoteActivityOptions}
 */
proto.temporal.omes.kitchen_sink.RemoteActivityOptions.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {!proto.temporal.omes.kitchen_sink.ActivityCancellationType} */ (reader.readEnum());
      msg.setCancellationType(value);
      break;
    case 2:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setDoNotEagerlyExecute(value);
      break;
    case 3:
      var value = /** @type {!proto.temporal.omes.kitchen_sink.VersioningIntent} */ (reader.readEnum());
      msg.setVersioningIntent(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.RemoteActivityOptions.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.RemoteActivityOptions.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.RemoteActivityOptions} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.RemoteActivityOptions.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCancellationType();
  if (f !== 0.0) {
    writer.writeEnum(
      1,
      f
    );
  }
  f = message.getDoNotEagerlyExecute();
  if (f) {
    writer.writeBool(
      2,
      f
    );
  }
  f = message.getVersioningIntent();
  if (f !== 0.0) {
    writer.writeEnum(
      3,
      f
    );
  }
};


/**
 * optional ActivityCancellationType cancellation_type = 1;
 * @return {!proto.temporal.omes.kitchen_sink.ActivityCancellationType}
 */
proto.temporal.omes.kitchen_sink.RemoteActivityOptions.prototype.getCancellationType = function() {
  return /** @type {!proto.temporal.omes.kitchen_sink.ActivityCancellationType} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {!proto.temporal.omes.kitchen_sink.ActivityCancellationType} value
 * @return {!proto.temporal.omes.kitchen_sink.RemoteActivityOptions} returns this
 */
proto.temporal.omes.kitchen_sink.RemoteActivityOptions.prototype.setCancellationType = function(value) {
  return jspb.Message.setProto3EnumField(this, 1, value);
};


/**
 * optional bool do_not_eagerly_execute = 2;
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.RemoteActivityOptions.prototype.getDoNotEagerlyExecute = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 2, false));
};


/**
 * @param {boolean} value
 * @return {!proto.temporal.omes.kitchen_sink.RemoteActivityOptions} returns this
 */
proto.temporal.omes.kitchen_sink.RemoteActivityOptions.prototype.setDoNotEagerlyExecute = function(value) {
  return jspb.Message.setProto3BooleanField(this, 2, value);
};


/**
 * optional VersioningIntent versioning_intent = 3;
 * @return {!proto.temporal.omes.kitchen_sink.VersioningIntent}
 */
proto.temporal.omes.kitchen_sink.RemoteActivityOptions.prototype.getVersioningIntent = function() {
  return /** @type {!proto.temporal.omes.kitchen_sink.VersioningIntent} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {!proto.temporal.omes.kitchen_sink.VersioningIntent} value
 * @return {!proto.temporal.omes.kitchen_sink.RemoteActivityOptions} returns this
 */
proto.temporal.omes.kitchen_sink.RemoteActivityOptions.prototype.setVersioningIntent = function(value) {
  return jspb.Message.setProto3EnumField(this, 3, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.temporal.omes.kitchen_sink.ExecuteNexusOperation.prototype.toObject = function(opt_includeInstance) {
  return proto.temporal.omes.kitchen_sink.ExecuteNexusOperation.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.temporal.omes.kitchen_sink.ExecuteNexusOperation} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.ExecuteNexusOperation.toObject = function(includeInstance, msg) {
  var f, obj = {
endpoint: jspb.Message.getFieldWithDefault(msg, 1, ""),
operation: jspb.Message.getFieldWithDefault(msg, 2, ""),
input: jspb.Message.getFieldWithDefault(msg, 3, ""),
headersMap: (f = msg.getHeadersMap()) ? f.toObject(includeInstance, undefined) : [],
awaitableChoice: (f = msg.getAwaitableChoice()) && proto.temporal.omes.kitchen_sink.AwaitableChoice.toObject(includeInstance, f),
expectedOutput: jspb.Message.getFieldWithDefault(msg, 6, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteNexusOperation}
 */
proto.temporal.omes.kitchen_sink.ExecuteNexusOperation.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.temporal.omes.kitchen_sink.ExecuteNexusOperation;
  return proto.temporal.omes.kitchen_sink.ExecuteNexusOperation.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.temporal.omes.kitchen_sink.ExecuteNexusOperation} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteNexusOperation}
 */
proto.temporal.omes.kitchen_sink.ExecuteNexusOperation.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setEndpoint(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setOperation(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setInput(value);
      break;
    case 4:
      var value = msg.getHeadersMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readString, null, "", "");
         });
      break;
    case 5:
      var value = new proto.temporal.omes.kitchen_sink.AwaitableChoice;
      reader.readMessage(value,proto.temporal.omes.kitchen_sink.AwaitableChoice.deserializeBinaryFromReader);
      msg.setAwaitableChoice(value);
      break;
    case 6:
      var value = /** @type {string} */ (reader.readString());
      msg.setExpectedOutput(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.temporal.omes.kitchen_sink.ExecuteNexusOperation.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.temporal.omes.kitchen_sink.ExecuteNexusOperation.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.temporal.omes.kitchen_sink.ExecuteNexusOperation} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.temporal.omes.kitchen_sink.ExecuteNexusOperation.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getEndpoint();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getOperation();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getInput();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getHeadersMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(4, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeString);
  }
  f = message.getAwaitableChoice();
  if (f != null) {
    writer.writeMessage(
      5,
      f,
      proto.temporal.omes.kitchen_sink.AwaitableChoice.serializeBinaryToWriter
    );
  }
  f = message.getExpectedOutput();
  if (f.length > 0) {
    writer.writeString(
      6,
      f
    );
  }
};


/**
 * optional string endpoint = 1;
 * @return {string}
 */
proto.temporal.omes.kitchen_sink.ExecuteNexusOperation.prototype.getEndpoint = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteNexusOperation} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteNexusOperation.prototype.setEndpoint = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string operation = 2;
 * @return {string}
 */
proto.temporal.omes.kitchen_sink.ExecuteNexusOperation.prototype.getOperation = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteNexusOperation} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteNexusOperation.prototype.setOperation = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string input = 3;
 * @return {string}
 */
proto.temporal.omes.kitchen_sink.ExecuteNexusOperation.prototype.getInput = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteNexusOperation} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteNexusOperation.prototype.setInput = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * map<string, string> headers = 4;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,string>}
 */
proto.temporal.omes.kitchen_sink.ExecuteNexusOperation.prototype.getHeadersMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,string>} */ (
      jspb.Message.getMapField(this, 4, opt_noLazyCreate,
      null));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteNexusOperation} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteNexusOperation.prototype.clearHeadersMap = function() {
  this.getHeadersMap().clear();
  return this;
};


/**
 * optional AwaitableChoice awaitable_choice = 5;
 * @return {?proto.temporal.omes.kitchen_sink.AwaitableChoice}
 */
proto.temporal.omes.kitchen_sink.ExecuteNexusOperation.prototype.getAwaitableChoice = function() {
  return /** @type{?proto.temporal.omes.kitchen_sink.AwaitableChoice} */ (
    jspb.Message.getWrapperField(this, proto.temporal.omes.kitchen_sink.AwaitableChoice, 5));
};


/**
 * @param {?proto.temporal.omes.kitchen_sink.AwaitableChoice|undefined} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteNexusOperation} returns this
*/
proto.temporal.omes.kitchen_sink.ExecuteNexusOperation.prototype.setAwaitableChoice = function(value) {
  return jspb.Message.setWrapperField(this, 5, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteNexusOperation} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteNexusOperation.prototype.clearAwaitableChoice = function() {
  return this.setAwaitableChoice(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.temporal.omes.kitchen_sink.ExecuteNexusOperation.prototype.hasAwaitableChoice = function() {
  return jspb.Message.getField(this, 5) != null;
};


/**
 * optional string expected_output = 6;
 * @return {string}
 */
proto.temporal.omes.kitchen_sink.ExecuteNexusOperation.prototype.getExpectedOutput = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 6, ""));
};


/**
 * @param {string} value
 * @return {!proto.temporal.omes.kitchen_sink.ExecuteNexusOperation} returns this
 */
proto.temporal.omes.kitchen_sink.ExecuteNexusOperation.prototype.setExpectedOutput = function(value) {
  return jspb.Message.setProto3StringField(this, 6, value);
};


/**
 * @enum {number}
 */
proto.temporal.omes.kitchen_sink.ParentClosePolicy = {
  PARENT_CLOSE_POLICY_UNSPECIFIED: 0,
  PARENT_CLOSE_POLICY_TERMINATE: 1,
  PARENT_CLOSE_POLICY_ABANDON: 2,
  PARENT_CLOSE_POLICY_REQUEST_CANCEL: 3
};

/**
 * @enum {number}
 */
proto.temporal.omes.kitchen_sink.VersioningIntent = {
  UNSPECIFIED: 0,
  COMPATIBLE: 1,
  DEFAULT: 2
};

/**
 * @enum {number}
 */
proto.temporal.omes.kitchen_sink.ChildWorkflowCancellationType = {
  CHILD_WF_ABANDON: 0,
  CHILD_WF_TRY_CANCEL: 1,
  CHILD_WF_WAIT_CANCELLATION_COMPLETED: 2,
  CHILD_WF_WAIT_CANCELLATION_REQUESTED: 3
};

/**
 * @enum {number}
 */
proto.temporal.omes.kitchen_sink.ActivityCancellationType = {
  TRY_CANCEL: 0,
  WAIT_CANCELLATION_COMPLETED: 1,
  ABANDON: 2
};

goog.object.extend(exports, proto.temporal.omes.kitchen_sink);
