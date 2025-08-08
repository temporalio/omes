// package: temporal.omes.kitchen_sink
// file: kitchen_sink/kitchen_sink.proto

import * as jspb from "google-protobuf";
import * as google_protobuf_duration_pb from "google-protobuf/google/protobuf/duration_pb";
import * as google_protobuf_empty_pb from "google-protobuf/google/protobuf/empty_pb";
import * as temporal_api_common_v1_message_pb from "../temporal/api/common/v1/message_pb";
import * as temporal_api_failure_v1_message_pb from "../temporal/api/failure/v1/message_pb";
import * as temporal_api_enums_v1_workflow_pb from "../temporal/api/enums/v1/workflow_pb";

export class TestInput extends jspb.Message {
  hasWorkflowInput(): boolean;
  clearWorkflowInput(): void;
  getWorkflowInput(): WorkflowInput | undefined;
  setWorkflowInput(value?: WorkflowInput): void;

  hasClientSequence(): boolean;
  clearClientSequence(): void;
  getClientSequence(): ClientSequence | undefined;
  setClientSequence(value?: ClientSequence): void;

  hasWithStartAction(): boolean;
  clearWithStartAction(): void;
  getWithStartAction(): WithStartClientAction | undefined;
  setWithStartAction(value?: WithStartClientAction): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TestInput.AsObject;
  static toObject(includeInstance: boolean, msg: TestInput): TestInput.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TestInput, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TestInput;
  static deserializeBinaryFromReader(message: TestInput, reader: jspb.BinaryReader): TestInput;
}

export namespace TestInput {
  export type AsObject = {
    workflowInput?: WorkflowInput.AsObject,
    clientSequence?: ClientSequence.AsObject,
    withStartAction?: WithStartClientAction.AsObject,
  }
}

export class ClientSequence extends jspb.Message {
  clearActionSetsList(): void;
  getActionSetsList(): Array<ClientActionSet>;
  setActionSetsList(value: Array<ClientActionSet>): void;
  addActionSets(value?: ClientActionSet, index?: number): ClientActionSet;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ClientSequence.AsObject;
  static toObject(includeInstance: boolean, msg: ClientSequence): ClientSequence.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ClientSequence, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ClientSequence;
  static deserializeBinaryFromReader(message: ClientSequence, reader: jspb.BinaryReader): ClientSequence;
}

export namespace ClientSequence {
  export type AsObject = {
    actionSetsList: Array<ClientActionSet.AsObject>,
  }
}

export class ClientActionSet extends jspb.Message {
  clearActionsList(): void;
  getActionsList(): Array<ClientAction>;
  setActionsList(value: Array<ClientAction>): void;
  addActions(value?: ClientAction, index?: number): ClientAction;

  getConcurrent(): boolean;
  setConcurrent(value: boolean): void;

  hasWaitAtEnd(): boolean;
  clearWaitAtEnd(): void;
  getWaitAtEnd(): google_protobuf_duration_pb.Duration | undefined;
  setWaitAtEnd(value?: google_protobuf_duration_pb.Duration): void;

  getWaitForCurrentRunToFinishAtEnd(): boolean;
  setWaitForCurrentRunToFinishAtEnd(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ClientActionSet.AsObject;
  static toObject(includeInstance: boolean, msg: ClientActionSet): ClientActionSet.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ClientActionSet, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ClientActionSet;
  static deserializeBinaryFromReader(message: ClientActionSet, reader: jspb.BinaryReader): ClientActionSet;
}

export namespace ClientActionSet {
  export type AsObject = {
    actionsList: Array<ClientAction.AsObject>,
    concurrent: boolean,
    waitAtEnd?: google_protobuf_duration_pb.Duration.AsObject,
    waitForCurrentRunToFinishAtEnd: boolean,
  }
}

export class WithStartClientAction extends jspb.Message {
  hasDoSignal(): boolean;
  clearDoSignal(): void;
  getDoSignal(): DoSignal | undefined;
  setDoSignal(value?: DoSignal): void;

  hasDoUpdate(): boolean;
  clearDoUpdate(): void;
  getDoUpdate(): DoUpdate | undefined;
  setDoUpdate(value?: DoUpdate): void;

  getVariantCase(): WithStartClientAction.VariantCase;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): WithStartClientAction.AsObject;
  static toObject(includeInstance: boolean, msg: WithStartClientAction): WithStartClientAction.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: WithStartClientAction, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): WithStartClientAction;
  static deserializeBinaryFromReader(message: WithStartClientAction, reader: jspb.BinaryReader): WithStartClientAction;
}

export namespace WithStartClientAction {
  export type AsObject = {
    doSignal?: DoSignal.AsObject,
    doUpdate?: DoUpdate.AsObject,
  }

  export enum VariantCase {
    VARIANT_NOT_SET = 0,
    DO_SIGNAL = 1,
    DO_UPDATE = 2,
  }
}

export class ClientAction extends jspb.Message {
  hasDoSignal(): boolean;
  clearDoSignal(): void;
  getDoSignal(): DoSignal | undefined;
  setDoSignal(value?: DoSignal): void;

  hasDoQuery(): boolean;
  clearDoQuery(): void;
  getDoQuery(): DoQuery | undefined;
  setDoQuery(value?: DoQuery): void;

  hasDoUpdate(): boolean;
  clearDoUpdate(): void;
  getDoUpdate(): DoUpdate | undefined;
  setDoUpdate(value?: DoUpdate): void;

  hasNestedActions(): boolean;
  clearNestedActions(): void;
  getNestedActions(): ClientActionSet | undefined;
  setNestedActions(value?: ClientActionSet): void;

  getVariantCase(): ClientAction.VariantCase;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ClientAction.AsObject;
  static toObject(includeInstance: boolean, msg: ClientAction): ClientAction.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ClientAction, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ClientAction;
  static deserializeBinaryFromReader(message: ClientAction, reader: jspb.BinaryReader): ClientAction;
}

export namespace ClientAction {
  export type AsObject = {
    doSignal?: DoSignal.AsObject,
    doQuery?: DoQuery.AsObject,
    doUpdate?: DoUpdate.AsObject,
    nestedActions?: ClientActionSet.AsObject,
  }

  export enum VariantCase {
    VARIANT_NOT_SET = 0,
    DO_SIGNAL = 1,
    DO_QUERY = 2,
    DO_UPDATE = 3,
    NESTED_ACTIONS = 4,
  }
}

export class DoSignal extends jspb.Message {
  hasDoSignalActions(): boolean;
  clearDoSignalActions(): void;
  getDoSignalActions(): DoSignal.DoSignalActions | undefined;
  setDoSignalActions(value?: DoSignal.DoSignalActions): void;

  hasCustom(): boolean;
  clearCustom(): void;
  getCustom(): HandlerInvocation | undefined;
  setCustom(value?: HandlerInvocation): void;

  getWithStart(): boolean;
  setWithStart(value: boolean): void;

  getVariantCase(): DoSignal.VariantCase;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DoSignal.AsObject;
  static toObject(includeInstance: boolean, msg: DoSignal): DoSignal.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DoSignal, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DoSignal;
  static deserializeBinaryFromReader(message: DoSignal, reader: jspb.BinaryReader): DoSignal;
}

export namespace DoSignal {
  export type AsObject = {
    doSignalActions?: DoSignal.DoSignalActions.AsObject,
    custom?: HandlerInvocation.AsObject,
    withStart: boolean,
  }

  export class DoSignalActions extends jspb.Message {
    hasDoActions(): boolean;
    clearDoActions(): void;
    getDoActions(): ActionSet | undefined;
    setDoActions(value?: ActionSet): void;

    hasDoActionsInMain(): boolean;
    clearDoActionsInMain(): void;
    getDoActionsInMain(): ActionSet | undefined;
    setDoActionsInMain(value?: ActionSet): void;

    getVariantCase(): DoSignalActions.VariantCase;
    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): DoSignalActions.AsObject;
    static toObject(includeInstance: boolean, msg: DoSignalActions): DoSignalActions.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: DoSignalActions, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): DoSignalActions;
    static deserializeBinaryFromReader(message: DoSignalActions, reader: jspb.BinaryReader): DoSignalActions;
  }

  export namespace DoSignalActions {
    export type AsObject = {
      doActions?: ActionSet.AsObject,
      doActionsInMain?: ActionSet.AsObject,
    }

    export enum VariantCase {
      VARIANT_NOT_SET = 0,
      DO_ACTIONS = 1,
      DO_ACTIONS_IN_MAIN = 2,
    }
  }

  export enum VariantCase {
    VARIANT_NOT_SET = 0,
    DO_SIGNAL_ACTIONS = 1,
    CUSTOM = 2,
  }
}

export class DoQuery extends jspb.Message {
  hasReportState(): boolean;
  clearReportState(): void;
  getReportState(): temporal_api_common_v1_message_pb.Payloads | undefined;
  setReportState(value?: temporal_api_common_v1_message_pb.Payloads): void;

  hasCustom(): boolean;
  clearCustom(): void;
  getCustom(): HandlerInvocation | undefined;
  setCustom(value?: HandlerInvocation): void;

  getFailureExpected(): boolean;
  setFailureExpected(value: boolean): void;

  getVariantCase(): DoQuery.VariantCase;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DoQuery.AsObject;
  static toObject(includeInstance: boolean, msg: DoQuery): DoQuery.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DoQuery, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DoQuery;
  static deserializeBinaryFromReader(message: DoQuery, reader: jspb.BinaryReader): DoQuery;
}

export namespace DoQuery {
  export type AsObject = {
    reportState?: temporal_api_common_v1_message_pb.Payloads.AsObject,
    custom?: HandlerInvocation.AsObject,
    failureExpected: boolean,
  }

  export enum VariantCase {
    VARIANT_NOT_SET = 0,
    REPORT_STATE = 1,
    CUSTOM = 2,
  }
}

export class DoUpdate extends jspb.Message {
  hasDoActions(): boolean;
  clearDoActions(): void;
  getDoActions(): DoActionsUpdate | undefined;
  setDoActions(value?: DoActionsUpdate): void;

  hasCustom(): boolean;
  clearCustom(): void;
  getCustom(): HandlerInvocation | undefined;
  setCustom(value?: HandlerInvocation): void;

  getWithStart(): boolean;
  setWithStart(value: boolean): void;

  getFailureExpected(): boolean;
  setFailureExpected(value: boolean): void;

  getVariantCase(): DoUpdate.VariantCase;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DoUpdate.AsObject;
  static toObject(includeInstance: boolean, msg: DoUpdate): DoUpdate.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DoUpdate, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DoUpdate;
  static deserializeBinaryFromReader(message: DoUpdate, reader: jspb.BinaryReader): DoUpdate;
}

export namespace DoUpdate {
  export type AsObject = {
    doActions?: DoActionsUpdate.AsObject,
    custom?: HandlerInvocation.AsObject,
    withStart: boolean,
    failureExpected: boolean,
  }

  export enum VariantCase {
    VARIANT_NOT_SET = 0,
    DO_ACTIONS = 1,
    CUSTOM = 2,
  }
}

export class DoActionsUpdate extends jspb.Message {
  hasDoActions(): boolean;
  clearDoActions(): void;
  getDoActions(): ActionSet | undefined;
  setDoActions(value?: ActionSet): void;

  hasRejectMe(): boolean;
  clearRejectMe(): void;
  getRejectMe(): google_protobuf_empty_pb.Empty | undefined;
  setRejectMe(value?: google_protobuf_empty_pb.Empty): void;

  getVariantCase(): DoActionsUpdate.VariantCase;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DoActionsUpdate.AsObject;
  static toObject(includeInstance: boolean, msg: DoActionsUpdate): DoActionsUpdate.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DoActionsUpdate, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DoActionsUpdate;
  static deserializeBinaryFromReader(message: DoActionsUpdate, reader: jspb.BinaryReader): DoActionsUpdate;
}

export namespace DoActionsUpdate {
  export type AsObject = {
    doActions?: ActionSet.AsObject,
    rejectMe?: google_protobuf_empty_pb.Empty.AsObject,
  }

  export enum VariantCase {
    VARIANT_NOT_SET = 0,
    DO_ACTIONS = 1,
    REJECT_ME = 2,
  }
}

export class HandlerInvocation extends jspb.Message {
  getName(): string;
  setName(value: string): void;

  clearArgsList(): void;
  getArgsList(): Array<temporal_api_common_v1_message_pb.Payload>;
  setArgsList(value: Array<temporal_api_common_v1_message_pb.Payload>): void;
  addArgs(value?: temporal_api_common_v1_message_pb.Payload, index?: number): temporal_api_common_v1_message_pb.Payload;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): HandlerInvocation.AsObject;
  static toObject(includeInstance: boolean, msg: HandlerInvocation): HandlerInvocation.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: HandlerInvocation, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): HandlerInvocation;
  static deserializeBinaryFromReader(message: HandlerInvocation, reader: jspb.BinaryReader): HandlerInvocation;
}

export namespace HandlerInvocation {
  export type AsObject = {
    name: string,
    argsList: Array<temporal_api_common_v1_message_pb.Payload.AsObject>,
  }
}

export class WorkflowState extends jspb.Message {
  getKvsMap(): jspb.Map<string, string>;
  clearKvsMap(): void;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): WorkflowState.AsObject;
  static toObject(includeInstance: boolean, msg: WorkflowState): WorkflowState.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: WorkflowState, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): WorkflowState;
  static deserializeBinaryFromReader(message: WorkflowState, reader: jspb.BinaryReader): WorkflowState;
}

export namespace WorkflowState {
  export type AsObject = {
    kvsMap: Array<[string, string]>,
  }
}

export class WorkflowInput extends jspb.Message {
  clearInitialActionsList(): void;
  getInitialActionsList(): Array<ActionSet>;
  setInitialActionsList(value: Array<ActionSet>): void;
  addInitialActions(value?: ActionSet, index?: number): ActionSet;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): WorkflowInput.AsObject;
  static toObject(includeInstance: boolean, msg: WorkflowInput): WorkflowInput.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: WorkflowInput, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): WorkflowInput;
  static deserializeBinaryFromReader(message: WorkflowInput, reader: jspb.BinaryReader): WorkflowInput;
}

export namespace WorkflowInput {
  export type AsObject = {
    initialActionsList: Array<ActionSet.AsObject>,
  }
}

export class ActionSet extends jspb.Message {
  clearActionsList(): void;
  getActionsList(): Array<Action>;
  setActionsList(value: Array<Action>): void;
  addActions(value?: Action, index?: number): Action;

  getConcurrent(): boolean;
  setConcurrent(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ActionSet.AsObject;
  static toObject(includeInstance: boolean, msg: ActionSet): ActionSet.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ActionSet, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ActionSet;
  static deserializeBinaryFromReader(message: ActionSet, reader: jspb.BinaryReader): ActionSet;
}

export namespace ActionSet {
  export type AsObject = {
    actionsList: Array<Action.AsObject>,
    concurrent: boolean,
  }
}

export class Action extends jspb.Message {
  hasTimer(): boolean;
  clearTimer(): void;
  getTimer(): TimerAction | undefined;
  setTimer(value?: TimerAction): void;

  hasExecActivity(): boolean;
  clearExecActivity(): void;
  getExecActivity(): ExecuteActivityAction | undefined;
  setExecActivity(value?: ExecuteActivityAction): void;

  hasExecChildWorkflow(): boolean;
  clearExecChildWorkflow(): void;
  getExecChildWorkflow(): ExecuteChildWorkflowAction | undefined;
  setExecChildWorkflow(value?: ExecuteChildWorkflowAction): void;

  hasAwaitWorkflowState(): boolean;
  clearAwaitWorkflowState(): void;
  getAwaitWorkflowState(): AwaitWorkflowState | undefined;
  setAwaitWorkflowState(value?: AwaitWorkflowState): void;

  hasSendSignal(): boolean;
  clearSendSignal(): void;
  getSendSignal(): SendSignalAction | undefined;
  setSendSignal(value?: SendSignalAction): void;

  hasCancelWorkflow(): boolean;
  clearCancelWorkflow(): void;
  getCancelWorkflow(): CancelWorkflowAction | undefined;
  setCancelWorkflow(value?: CancelWorkflowAction): void;

  hasSetPatchMarker(): boolean;
  clearSetPatchMarker(): void;
  getSetPatchMarker(): SetPatchMarkerAction | undefined;
  setSetPatchMarker(value?: SetPatchMarkerAction): void;

  hasUpsertSearchAttributes(): boolean;
  clearUpsertSearchAttributes(): void;
  getUpsertSearchAttributes(): UpsertSearchAttributesAction | undefined;
  setUpsertSearchAttributes(value?: UpsertSearchAttributesAction): void;

  hasUpsertMemo(): boolean;
  clearUpsertMemo(): void;
  getUpsertMemo(): UpsertMemoAction | undefined;
  setUpsertMemo(value?: UpsertMemoAction): void;

  hasSetWorkflowState(): boolean;
  clearSetWorkflowState(): void;
  getSetWorkflowState(): WorkflowState | undefined;
  setSetWorkflowState(value?: WorkflowState): void;

  hasReturnResult(): boolean;
  clearReturnResult(): void;
  getReturnResult(): ReturnResultAction | undefined;
  setReturnResult(value?: ReturnResultAction): void;

  hasReturnError(): boolean;
  clearReturnError(): void;
  getReturnError(): ReturnErrorAction | undefined;
  setReturnError(value?: ReturnErrorAction): void;

  hasContinueAsNew(): boolean;
  clearContinueAsNew(): void;
  getContinueAsNew(): ContinueAsNewAction | undefined;
  setContinueAsNew(value?: ContinueAsNewAction): void;

  hasNestedActionSet(): boolean;
  clearNestedActionSet(): void;
  getNestedActionSet(): ActionSet | undefined;
  setNestedActionSet(value?: ActionSet): void;

  hasNexusOperation(): boolean;
  clearNexusOperation(): void;
  getNexusOperation(): ExecuteNexusOperation | undefined;
  setNexusOperation(value?: ExecuteNexusOperation): void;

  getVariantCase(): Action.VariantCase;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Action.AsObject;
  static toObject(includeInstance: boolean, msg: Action): Action.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Action, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Action;
  static deserializeBinaryFromReader(message: Action, reader: jspb.BinaryReader): Action;
}

export namespace Action {
  export type AsObject = {
    timer?: TimerAction.AsObject,
    execActivity?: ExecuteActivityAction.AsObject,
    execChildWorkflow?: ExecuteChildWorkflowAction.AsObject,
    awaitWorkflowState?: AwaitWorkflowState.AsObject,
    sendSignal?: SendSignalAction.AsObject,
    cancelWorkflow?: CancelWorkflowAction.AsObject,
    setPatchMarker?: SetPatchMarkerAction.AsObject,
    upsertSearchAttributes?: UpsertSearchAttributesAction.AsObject,
    upsertMemo?: UpsertMemoAction.AsObject,
    setWorkflowState?: WorkflowState.AsObject,
    returnResult?: ReturnResultAction.AsObject,
    returnError?: ReturnErrorAction.AsObject,
    continueAsNew?: ContinueAsNewAction.AsObject,
    nestedActionSet?: ActionSet.AsObject,
    nexusOperation?: ExecuteNexusOperation.AsObject,
  }

  export enum VariantCase {
    VARIANT_NOT_SET = 0,
    TIMER = 1,
    EXEC_ACTIVITY = 2,
    EXEC_CHILD_WORKFLOW = 3,
    AWAIT_WORKFLOW_STATE = 4,
    SEND_SIGNAL = 5,
    CANCEL_WORKFLOW = 6,
    SET_PATCH_MARKER = 7,
    UPSERT_SEARCH_ATTRIBUTES = 8,
    UPSERT_MEMO = 9,
    SET_WORKFLOW_STATE = 10,
    RETURN_RESULT = 11,
    RETURN_ERROR = 12,
    CONTINUE_AS_NEW = 13,
    NESTED_ACTION_SET = 14,
    NEXUS_OPERATION = 15,
  }
}

export class AwaitableChoice extends jspb.Message {
  hasWaitFinish(): boolean;
  clearWaitFinish(): void;
  getWaitFinish(): google_protobuf_empty_pb.Empty | undefined;
  setWaitFinish(value?: google_protobuf_empty_pb.Empty): void;

  hasAbandon(): boolean;
  clearAbandon(): void;
  getAbandon(): google_protobuf_empty_pb.Empty | undefined;
  setAbandon(value?: google_protobuf_empty_pb.Empty): void;

  hasCancelBeforeStarted(): boolean;
  clearCancelBeforeStarted(): void;
  getCancelBeforeStarted(): google_protobuf_empty_pb.Empty | undefined;
  setCancelBeforeStarted(value?: google_protobuf_empty_pb.Empty): void;

  hasCancelAfterStarted(): boolean;
  clearCancelAfterStarted(): void;
  getCancelAfterStarted(): google_protobuf_empty_pb.Empty | undefined;
  setCancelAfterStarted(value?: google_protobuf_empty_pb.Empty): void;

  hasCancelAfterCompleted(): boolean;
  clearCancelAfterCompleted(): void;
  getCancelAfterCompleted(): google_protobuf_empty_pb.Empty | undefined;
  setCancelAfterCompleted(value?: google_protobuf_empty_pb.Empty): void;

  getConditionCase(): AwaitableChoice.ConditionCase;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AwaitableChoice.AsObject;
  static toObject(includeInstance: boolean, msg: AwaitableChoice): AwaitableChoice.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: AwaitableChoice, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AwaitableChoice;
  static deserializeBinaryFromReader(message: AwaitableChoice, reader: jspb.BinaryReader): AwaitableChoice;
}

export namespace AwaitableChoice {
  export type AsObject = {
    waitFinish?: google_protobuf_empty_pb.Empty.AsObject,
    abandon?: google_protobuf_empty_pb.Empty.AsObject,
    cancelBeforeStarted?: google_protobuf_empty_pb.Empty.AsObject,
    cancelAfterStarted?: google_protobuf_empty_pb.Empty.AsObject,
    cancelAfterCompleted?: google_protobuf_empty_pb.Empty.AsObject,
  }

  export enum ConditionCase {
    CONDITION_NOT_SET = 0,
    WAIT_FINISH = 1,
    ABANDON = 2,
    CANCEL_BEFORE_STARTED = 3,
    CANCEL_AFTER_STARTED = 4,
    CANCEL_AFTER_COMPLETED = 5,
  }
}

export class TimerAction extends jspb.Message {
  getMilliseconds(): number;
  setMilliseconds(value: number): void;

  hasAwaitableChoice(): boolean;
  clearAwaitableChoice(): void;
  getAwaitableChoice(): AwaitableChoice | undefined;
  setAwaitableChoice(value?: AwaitableChoice): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TimerAction.AsObject;
  static toObject(includeInstance: boolean, msg: TimerAction): TimerAction.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TimerAction, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TimerAction;
  static deserializeBinaryFromReader(message: TimerAction, reader: jspb.BinaryReader): TimerAction;
}

export namespace TimerAction {
  export type AsObject = {
    milliseconds: number,
    awaitableChoice?: AwaitableChoice.AsObject,
  }
}

export class ExecuteActivityAction extends jspb.Message {
  hasGeneric(): boolean;
  clearGeneric(): void;
  getGeneric(): ExecuteActivityAction.GenericActivity | undefined;
  setGeneric(value?: ExecuteActivityAction.GenericActivity): void;

  hasDelay(): boolean;
  clearDelay(): void;
  getDelay(): google_protobuf_duration_pb.Duration | undefined;
  setDelay(value?: google_protobuf_duration_pb.Duration): void;

  hasNoop(): boolean;
  clearNoop(): void;
  getNoop(): google_protobuf_empty_pb.Empty | undefined;
  setNoop(value?: google_protobuf_empty_pb.Empty): void;

  hasResources(): boolean;
  clearResources(): void;
  getResources(): ExecuteActivityAction.ResourcesActivity | undefined;
  setResources(value?: ExecuteActivityAction.ResourcesActivity): void;

  hasPayload(): boolean;
  clearPayload(): void;
  getPayload(): ExecuteActivityAction.PayloadActivity | undefined;
  setPayload(value?: ExecuteActivityAction.PayloadActivity): void;

  getTaskQueue(): string;
  setTaskQueue(value: string): void;

  getHeadersMap(): jspb.Map<string, temporal_api_common_v1_message_pb.Payload>;
  clearHeadersMap(): void;
  hasScheduleToCloseTimeout(): boolean;
  clearScheduleToCloseTimeout(): void;
  getScheduleToCloseTimeout(): google_protobuf_duration_pb.Duration | undefined;
  setScheduleToCloseTimeout(value?: google_protobuf_duration_pb.Duration): void;

  hasScheduleToStartTimeout(): boolean;
  clearScheduleToStartTimeout(): void;
  getScheduleToStartTimeout(): google_protobuf_duration_pb.Duration | undefined;
  setScheduleToStartTimeout(value?: google_protobuf_duration_pb.Duration): void;

  hasStartToCloseTimeout(): boolean;
  clearStartToCloseTimeout(): void;
  getStartToCloseTimeout(): google_protobuf_duration_pb.Duration | undefined;
  setStartToCloseTimeout(value?: google_protobuf_duration_pb.Duration): void;

  hasHeartbeatTimeout(): boolean;
  clearHeartbeatTimeout(): void;
  getHeartbeatTimeout(): google_protobuf_duration_pb.Duration | undefined;
  setHeartbeatTimeout(value?: google_protobuf_duration_pb.Duration): void;

  hasRetryPolicy(): boolean;
  clearRetryPolicy(): void;
  getRetryPolicy(): temporal_api_common_v1_message_pb.RetryPolicy | undefined;
  setRetryPolicy(value?: temporal_api_common_v1_message_pb.RetryPolicy): void;

  hasIsLocal(): boolean;
  clearIsLocal(): void;
  getIsLocal(): google_protobuf_empty_pb.Empty | undefined;
  setIsLocal(value?: google_protobuf_empty_pb.Empty): void;

  hasRemote(): boolean;
  clearRemote(): void;
  getRemote(): RemoteActivityOptions | undefined;
  setRemote(value?: RemoteActivityOptions): void;

  hasAwaitableChoice(): boolean;
  clearAwaitableChoice(): void;
  getAwaitableChoice(): AwaitableChoice | undefined;
  setAwaitableChoice(value?: AwaitableChoice): void;

  hasPriority(): boolean;
  clearPriority(): void;
  getPriority(): temporal_api_common_v1_message_pb.Priority | undefined;
  setPriority(value?: temporal_api_common_v1_message_pb.Priority): void;

  getFairnessKey(): string;
  setFairnessKey(value: string): void;

  getFairnessWeight(): number;
  setFairnessWeight(value: number): void;

  getActivityTypeCase(): ExecuteActivityAction.ActivityTypeCase;
  getLocalityCase(): ExecuteActivityAction.LocalityCase;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ExecuteActivityAction.AsObject;
  static toObject(includeInstance: boolean, msg: ExecuteActivityAction): ExecuteActivityAction.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ExecuteActivityAction, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ExecuteActivityAction;
  static deserializeBinaryFromReader(message: ExecuteActivityAction, reader: jspb.BinaryReader): ExecuteActivityAction;
}

export namespace ExecuteActivityAction {
  export type AsObject = {
    generic?: ExecuteActivityAction.GenericActivity.AsObject,
    delay?: google_protobuf_duration_pb.Duration.AsObject,
    noop?: google_protobuf_empty_pb.Empty.AsObject,
    resources?: ExecuteActivityAction.ResourcesActivity.AsObject,
    payload?: ExecuteActivityAction.PayloadActivity.AsObject,
    taskQueue: string,
    headersMap: Array<[string, temporal_api_common_v1_message_pb.Payload.AsObject]>,
    scheduleToCloseTimeout?: google_protobuf_duration_pb.Duration.AsObject,
    scheduleToStartTimeout?: google_protobuf_duration_pb.Duration.AsObject,
    startToCloseTimeout?: google_protobuf_duration_pb.Duration.AsObject,
    heartbeatTimeout?: google_protobuf_duration_pb.Duration.AsObject,
    retryPolicy?: temporal_api_common_v1_message_pb.RetryPolicy.AsObject,
    isLocal?: google_protobuf_empty_pb.Empty.AsObject,
    remote?: RemoteActivityOptions.AsObject,
    awaitableChoice?: AwaitableChoice.AsObject,
    priority?: temporal_api_common_v1_message_pb.Priority.AsObject,
    fairnessKey: string,
    fairnessWeight: number,
  }

  export class GenericActivity extends jspb.Message {
    getType(): string;
    setType(value: string): void;

    clearArgumentsList(): void;
    getArgumentsList(): Array<temporal_api_common_v1_message_pb.Payload>;
    setArgumentsList(value: Array<temporal_api_common_v1_message_pb.Payload>): void;
    addArguments(value?: temporal_api_common_v1_message_pb.Payload, index?: number): temporal_api_common_v1_message_pb.Payload;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): GenericActivity.AsObject;
    static toObject(includeInstance: boolean, msg: GenericActivity): GenericActivity.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: GenericActivity, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): GenericActivity;
    static deserializeBinaryFromReader(message: GenericActivity, reader: jspb.BinaryReader): GenericActivity;
  }

  export namespace GenericActivity {
    export type AsObject = {
      type: string,
      argumentsList: Array<temporal_api_common_v1_message_pb.Payload.AsObject>,
    }
  }

  export class ResourcesActivity extends jspb.Message {
    hasRunFor(): boolean;
    clearRunFor(): void;
    getRunFor(): google_protobuf_duration_pb.Duration | undefined;
    setRunFor(value?: google_protobuf_duration_pb.Duration): void;

    getBytesToAllocate(): number;
    setBytesToAllocate(value: number): void;

    getCpuYieldEveryNIterations(): number;
    setCpuYieldEveryNIterations(value: number): void;

    getCpuYieldForMs(): number;
    setCpuYieldForMs(value: number): void;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ResourcesActivity.AsObject;
    static toObject(includeInstance: boolean, msg: ResourcesActivity): ResourcesActivity.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: ResourcesActivity, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ResourcesActivity;
    static deserializeBinaryFromReader(message: ResourcesActivity, reader: jspb.BinaryReader): ResourcesActivity;
  }

  export namespace ResourcesActivity {
    export type AsObject = {
      runFor?: google_protobuf_duration_pb.Duration.AsObject,
      bytesToAllocate: number,
      cpuYieldEveryNIterations: number,
      cpuYieldForMs: number,
    }
  }

  export class PayloadActivity extends jspb.Message {
    getBytesToReceive(): number;
    setBytesToReceive(value: number): void;

    getBytesToReturn(): number;
    setBytesToReturn(value: number): void;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): PayloadActivity.AsObject;
    static toObject(includeInstance: boolean, msg: PayloadActivity): PayloadActivity.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: PayloadActivity, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): PayloadActivity;
    static deserializeBinaryFromReader(message: PayloadActivity, reader: jspb.BinaryReader): PayloadActivity;
  }

  export namespace PayloadActivity {
    export type AsObject = {
      bytesToReceive: number,
      bytesToReturn: number,
    }
  }

  export enum ActivityTypeCase {
    ACTIVITY_TYPE_NOT_SET = 0,
    GENERIC = 1,
    DELAY = 2,
    NOOP = 3,
    RESOURCES = 14,
    PAYLOAD = 18,
  }

  export enum LocalityCase {
    LOCALITY_NOT_SET = 0,
    IS_LOCAL = 11,
    REMOTE = 12,
  }
}

export class ExecuteChildWorkflowAction extends jspb.Message {
  getNamespace(): string;
  setNamespace(value: string): void;

  getWorkflowId(): string;
  setWorkflowId(value: string): void;

  getWorkflowType(): string;
  setWorkflowType(value: string): void;

  getTaskQueue(): string;
  setTaskQueue(value: string): void;

  clearInputList(): void;
  getInputList(): Array<temporal_api_common_v1_message_pb.Payload>;
  setInputList(value: Array<temporal_api_common_v1_message_pb.Payload>): void;
  addInput(value?: temporal_api_common_v1_message_pb.Payload, index?: number): temporal_api_common_v1_message_pb.Payload;

  hasWorkflowExecutionTimeout(): boolean;
  clearWorkflowExecutionTimeout(): void;
  getWorkflowExecutionTimeout(): google_protobuf_duration_pb.Duration | undefined;
  setWorkflowExecutionTimeout(value?: google_protobuf_duration_pb.Duration): void;

  hasWorkflowRunTimeout(): boolean;
  clearWorkflowRunTimeout(): void;
  getWorkflowRunTimeout(): google_protobuf_duration_pb.Duration | undefined;
  setWorkflowRunTimeout(value?: google_protobuf_duration_pb.Duration): void;

  hasWorkflowTaskTimeout(): boolean;
  clearWorkflowTaskTimeout(): void;
  getWorkflowTaskTimeout(): google_protobuf_duration_pb.Duration | undefined;
  setWorkflowTaskTimeout(value?: google_protobuf_duration_pb.Duration): void;

  getParentClosePolicy(): ParentClosePolicyMap[keyof ParentClosePolicyMap];
  setParentClosePolicy(value: ParentClosePolicyMap[keyof ParentClosePolicyMap]): void;

  getWorkflowIdReusePolicy(): temporal_api_enums_v1_workflow_pb.WorkflowIdReusePolicyMap[keyof temporal_api_enums_v1_workflow_pb.WorkflowIdReusePolicyMap];
  setWorkflowIdReusePolicy(value: temporal_api_enums_v1_workflow_pb.WorkflowIdReusePolicyMap[keyof temporal_api_enums_v1_workflow_pb.WorkflowIdReusePolicyMap]): void;

  hasRetryPolicy(): boolean;
  clearRetryPolicy(): void;
  getRetryPolicy(): temporal_api_common_v1_message_pb.RetryPolicy | undefined;
  setRetryPolicy(value?: temporal_api_common_v1_message_pb.RetryPolicy): void;

  getCronSchedule(): string;
  setCronSchedule(value: string): void;

  getHeadersMap(): jspb.Map<string, temporal_api_common_v1_message_pb.Payload>;
  clearHeadersMap(): void;
  getMemoMap(): jspb.Map<string, temporal_api_common_v1_message_pb.Payload>;
  clearMemoMap(): void;
  getSearchAttributesMap(): jspb.Map<string, temporal_api_common_v1_message_pb.Payload>;
  clearSearchAttributesMap(): void;
  getCancellationType(): ChildWorkflowCancellationTypeMap[keyof ChildWorkflowCancellationTypeMap];
  setCancellationType(value: ChildWorkflowCancellationTypeMap[keyof ChildWorkflowCancellationTypeMap]): void;

  getVersioningIntent(): VersioningIntentMap[keyof VersioningIntentMap];
  setVersioningIntent(value: VersioningIntentMap[keyof VersioningIntentMap]): void;

  hasAwaitableChoice(): boolean;
  clearAwaitableChoice(): void;
  getAwaitableChoice(): AwaitableChoice | undefined;
  setAwaitableChoice(value?: AwaitableChoice): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ExecuteChildWorkflowAction.AsObject;
  static toObject(includeInstance: boolean, msg: ExecuteChildWorkflowAction): ExecuteChildWorkflowAction.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ExecuteChildWorkflowAction, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ExecuteChildWorkflowAction;
  static deserializeBinaryFromReader(message: ExecuteChildWorkflowAction, reader: jspb.BinaryReader): ExecuteChildWorkflowAction;
}

export namespace ExecuteChildWorkflowAction {
  export type AsObject = {
    namespace: string,
    workflowId: string,
    workflowType: string,
    taskQueue: string,
    inputList: Array<temporal_api_common_v1_message_pb.Payload.AsObject>,
    workflowExecutionTimeout?: google_protobuf_duration_pb.Duration.AsObject,
    workflowRunTimeout?: google_protobuf_duration_pb.Duration.AsObject,
    workflowTaskTimeout?: google_protobuf_duration_pb.Duration.AsObject,
    parentClosePolicy: ParentClosePolicyMap[keyof ParentClosePolicyMap],
    workflowIdReusePolicy: temporal_api_enums_v1_workflow_pb.WorkflowIdReusePolicyMap[keyof temporal_api_enums_v1_workflow_pb.WorkflowIdReusePolicyMap],
    retryPolicy?: temporal_api_common_v1_message_pb.RetryPolicy.AsObject,
    cronSchedule: string,
    headersMap: Array<[string, temporal_api_common_v1_message_pb.Payload.AsObject]>,
    memoMap: Array<[string, temporal_api_common_v1_message_pb.Payload.AsObject]>,
    searchAttributesMap: Array<[string, temporal_api_common_v1_message_pb.Payload.AsObject]>,
    cancellationType: ChildWorkflowCancellationTypeMap[keyof ChildWorkflowCancellationTypeMap],
    versioningIntent: VersioningIntentMap[keyof VersioningIntentMap],
    awaitableChoice?: AwaitableChoice.AsObject,
  }
}

export class AwaitWorkflowState extends jspb.Message {
  getKey(): string;
  setKey(value: string): void;

  getValue(): string;
  setValue(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AwaitWorkflowState.AsObject;
  static toObject(includeInstance: boolean, msg: AwaitWorkflowState): AwaitWorkflowState.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: AwaitWorkflowState, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AwaitWorkflowState;
  static deserializeBinaryFromReader(message: AwaitWorkflowState, reader: jspb.BinaryReader): AwaitWorkflowState;
}

export namespace AwaitWorkflowState {
  export type AsObject = {
    key: string,
    value: string,
  }
}

export class SendSignalAction extends jspb.Message {
  getWorkflowId(): string;
  setWorkflowId(value: string): void;

  getRunId(): string;
  setRunId(value: string): void;

  getSignalName(): string;
  setSignalName(value: string): void;

  clearArgsList(): void;
  getArgsList(): Array<temporal_api_common_v1_message_pb.Payload>;
  setArgsList(value: Array<temporal_api_common_v1_message_pb.Payload>): void;
  addArgs(value?: temporal_api_common_v1_message_pb.Payload, index?: number): temporal_api_common_v1_message_pb.Payload;

  getHeadersMap(): jspb.Map<string, temporal_api_common_v1_message_pb.Payload>;
  clearHeadersMap(): void;
  hasAwaitableChoice(): boolean;
  clearAwaitableChoice(): void;
  getAwaitableChoice(): AwaitableChoice | undefined;
  setAwaitableChoice(value?: AwaitableChoice): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SendSignalAction.AsObject;
  static toObject(includeInstance: boolean, msg: SendSignalAction): SendSignalAction.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SendSignalAction, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SendSignalAction;
  static deserializeBinaryFromReader(message: SendSignalAction, reader: jspb.BinaryReader): SendSignalAction;
}

export namespace SendSignalAction {
  export type AsObject = {
    workflowId: string,
    runId: string,
    signalName: string,
    argsList: Array<temporal_api_common_v1_message_pb.Payload.AsObject>,
    headersMap: Array<[string, temporal_api_common_v1_message_pb.Payload.AsObject]>,
    awaitableChoice?: AwaitableChoice.AsObject,
  }
}

export class CancelWorkflowAction extends jspb.Message {
  getWorkflowId(): string;
  setWorkflowId(value: string): void;

  getRunId(): string;
  setRunId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CancelWorkflowAction.AsObject;
  static toObject(includeInstance: boolean, msg: CancelWorkflowAction): CancelWorkflowAction.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: CancelWorkflowAction, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CancelWorkflowAction;
  static deserializeBinaryFromReader(message: CancelWorkflowAction, reader: jspb.BinaryReader): CancelWorkflowAction;
}

export namespace CancelWorkflowAction {
  export type AsObject = {
    workflowId: string,
    runId: string,
  }
}

export class SetPatchMarkerAction extends jspb.Message {
  getPatchId(): string;
  setPatchId(value: string): void;

  getDeprecated(): boolean;
  setDeprecated(value: boolean): void;

  hasInnerAction(): boolean;
  clearInnerAction(): void;
  getInnerAction(): Action | undefined;
  setInnerAction(value?: Action): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SetPatchMarkerAction.AsObject;
  static toObject(includeInstance: boolean, msg: SetPatchMarkerAction): SetPatchMarkerAction.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SetPatchMarkerAction, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SetPatchMarkerAction;
  static deserializeBinaryFromReader(message: SetPatchMarkerAction, reader: jspb.BinaryReader): SetPatchMarkerAction;
}

export namespace SetPatchMarkerAction {
  export type AsObject = {
    patchId: string,
    deprecated: boolean,
    innerAction?: Action.AsObject,
  }
}

export class UpsertSearchAttributesAction extends jspb.Message {
  getSearchAttributesMap(): jspb.Map<string, temporal_api_common_v1_message_pb.Payload>;
  clearSearchAttributesMap(): void;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpsertSearchAttributesAction.AsObject;
  static toObject(includeInstance: boolean, msg: UpsertSearchAttributesAction): UpsertSearchAttributesAction.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: UpsertSearchAttributesAction, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpsertSearchAttributesAction;
  static deserializeBinaryFromReader(message: UpsertSearchAttributesAction, reader: jspb.BinaryReader): UpsertSearchAttributesAction;
}

export namespace UpsertSearchAttributesAction {
  export type AsObject = {
    searchAttributesMap: Array<[string, temporal_api_common_v1_message_pb.Payload.AsObject]>,
  }
}

export class UpsertMemoAction extends jspb.Message {
  hasUpsertedMemo(): boolean;
  clearUpsertedMemo(): void;
  getUpsertedMemo(): temporal_api_common_v1_message_pb.Memo | undefined;
  setUpsertedMemo(value?: temporal_api_common_v1_message_pb.Memo): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpsertMemoAction.AsObject;
  static toObject(includeInstance: boolean, msg: UpsertMemoAction): UpsertMemoAction.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: UpsertMemoAction, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpsertMemoAction;
  static deserializeBinaryFromReader(message: UpsertMemoAction, reader: jspb.BinaryReader): UpsertMemoAction;
}

export namespace UpsertMemoAction {
  export type AsObject = {
    upsertedMemo?: temporal_api_common_v1_message_pb.Memo.AsObject,
  }
}

export class ReturnResultAction extends jspb.Message {
  hasReturnThis(): boolean;
  clearReturnThis(): void;
  getReturnThis(): temporal_api_common_v1_message_pb.Payload | undefined;
  setReturnThis(value?: temporal_api_common_v1_message_pb.Payload): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ReturnResultAction.AsObject;
  static toObject(includeInstance: boolean, msg: ReturnResultAction): ReturnResultAction.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ReturnResultAction, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ReturnResultAction;
  static deserializeBinaryFromReader(message: ReturnResultAction, reader: jspb.BinaryReader): ReturnResultAction;
}

export namespace ReturnResultAction {
  export type AsObject = {
    returnThis?: temporal_api_common_v1_message_pb.Payload.AsObject,
  }
}

export class ReturnErrorAction extends jspb.Message {
  hasFailure(): boolean;
  clearFailure(): void;
  getFailure(): temporal_api_failure_v1_message_pb.Failure | undefined;
  setFailure(value?: temporal_api_failure_v1_message_pb.Failure): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ReturnErrorAction.AsObject;
  static toObject(includeInstance: boolean, msg: ReturnErrorAction): ReturnErrorAction.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ReturnErrorAction, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ReturnErrorAction;
  static deserializeBinaryFromReader(message: ReturnErrorAction, reader: jspb.BinaryReader): ReturnErrorAction;
}

export namespace ReturnErrorAction {
  export type AsObject = {
    failure?: temporal_api_failure_v1_message_pb.Failure.AsObject,
  }
}

export class ContinueAsNewAction extends jspb.Message {
  getWorkflowType(): string;
  setWorkflowType(value: string): void;

  getTaskQueue(): string;
  setTaskQueue(value: string): void;

  clearArgumentsList(): void;
  getArgumentsList(): Array<temporal_api_common_v1_message_pb.Payload>;
  setArgumentsList(value: Array<temporal_api_common_v1_message_pb.Payload>): void;
  addArguments(value?: temporal_api_common_v1_message_pb.Payload, index?: number): temporal_api_common_v1_message_pb.Payload;

  hasWorkflowRunTimeout(): boolean;
  clearWorkflowRunTimeout(): void;
  getWorkflowRunTimeout(): google_protobuf_duration_pb.Duration | undefined;
  setWorkflowRunTimeout(value?: google_protobuf_duration_pb.Duration): void;

  hasWorkflowTaskTimeout(): boolean;
  clearWorkflowTaskTimeout(): void;
  getWorkflowTaskTimeout(): google_protobuf_duration_pb.Duration | undefined;
  setWorkflowTaskTimeout(value?: google_protobuf_duration_pb.Duration): void;

  getMemoMap(): jspb.Map<string, temporal_api_common_v1_message_pb.Payload>;
  clearMemoMap(): void;
  getHeadersMap(): jspb.Map<string, temporal_api_common_v1_message_pb.Payload>;
  clearHeadersMap(): void;
  getSearchAttributesMap(): jspb.Map<string, temporal_api_common_v1_message_pb.Payload>;
  clearSearchAttributesMap(): void;
  hasRetryPolicy(): boolean;
  clearRetryPolicy(): void;
  getRetryPolicy(): temporal_api_common_v1_message_pb.RetryPolicy | undefined;
  setRetryPolicy(value?: temporal_api_common_v1_message_pb.RetryPolicy): void;

  getVersioningIntent(): VersioningIntentMap[keyof VersioningIntentMap];
  setVersioningIntent(value: VersioningIntentMap[keyof VersioningIntentMap]): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ContinueAsNewAction.AsObject;
  static toObject(includeInstance: boolean, msg: ContinueAsNewAction): ContinueAsNewAction.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ContinueAsNewAction, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ContinueAsNewAction;
  static deserializeBinaryFromReader(message: ContinueAsNewAction, reader: jspb.BinaryReader): ContinueAsNewAction;
}

export namespace ContinueAsNewAction {
  export type AsObject = {
    workflowType: string,
    taskQueue: string,
    argumentsList: Array<temporal_api_common_v1_message_pb.Payload.AsObject>,
    workflowRunTimeout?: google_protobuf_duration_pb.Duration.AsObject,
    workflowTaskTimeout?: google_protobuf_duration_pb.Duration.AsObject,
    memoMap: Array<[string, temporal_api_common_v1_message_pb.Payload.AsObject]>,
    headersMap: Array<[string, temporal_api_common_v1_message_pb.Payload.AsObject]>,
    searchAttributesMap: Array<[string, temporal_api_common_v1_message_pb.Payload.AsObject]>,
    retryPolicy?: temporal_api_common_v1_message_pb.RetryPolicy.AsObject,
    versioningIntent: VersioningIntentMap[keyof VersioningIntentMap],
  }
}

export class RemoteActivityOptions extends jspb.Message {
  getCancellationType(): ActivityCancellationTypeMap[keyof ActivityCancellationTypeMap];
  setCancellationType(value: ActivityCancellationTypeMap[keyof ActivityCancellationTypeMap]): void;

  getDoNotEagerlyExecute(): boolean;
  setDoNotEagerlyExecute(value: boolean): void;

  getVersioningIntent(): VersioningIntentMap[keyof VersioningIntentMap];
  setVersioningIntent(value: VersioningIntentMap[keyof VersioningIntentMap]): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RemoteActivityOptions.AsObject;
  static toObject(includeInstance: boolean, msg: RemoteActivityOptions): RemoteActivityOptions.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: RemoteActivityOptions, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RemoteActivityOptions;
  static deserializeBinaryFromReader(message: RemoteActivityOptions, reader: jspb.BinaryReader): RemoteActivityOptions;
}

export namespace RemoteActivityOptions {
  export type AsObject = {
    cancellationType: ActivityCancellationTypeMap[keyof ActivityCancellationTypeMap],
    doNotEagerlyExecute: boolean,
    versioningIntent: VersioningIntentMap[keyof VersioningIntentMap],
  }
}

export class ExecuteNexusOperation extends jspb.Message {
  getEndpoint(): string;
  setEndpoint(value: string): void;

  getOperation(): string;
  setOperation(value: string): void;

  getInput(): string;
  setInput(value: string): void;

  getHeadersMap(): jspb.Map<string, string>;
  clearHeadersMap(): void;
  hasAwaitableChoice(): boolean;
  clearAwaitableChoice(): void;
  getAwaitableChoice(): AwaitableChoice | undefined;
  setAwaitableChoice(value?: AwaitableChoice): void;

  getExpectedOutput(): string;
  setExpectedOutput(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ExecuteNexusOperation.AsObject;
  static toObject(includeInstance: boolean, msg: ExecuteNexusOperation): ExecuteNexusOperation.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ExecuteNexusOperation, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ExecuteNexusOperation;
  static deserializeBinaryFromReader(message: ExecuteNexusOperation, reader: jspb.BinaryReader): ExecuteNexusOperation;
}

export namespace ExecuteNexusOperation {
  export type AsObject = {
    endpoint: string,
    operation: string,
    input: string,
    headersMap: Array<[string, string]>,
    awaitableChoice?: AwaitableChoice.AsObject,
    expectedOutput: string,
  }
}

export interface ParentClosePolicyMap {
  PARENT_CLOSE_POLICY_UNSPECIFIED: 0;
  PARENT_CLOSE_POLICY_TERMINATE: 1;
  PARENT_CLOSE_POLICY_ABANDON: 2;
  PARENT_CLOSE_POLICY_REQUEST_CANCEL: 3;
}

export const ParentClosePolicy: ParentClosePolicyMap;

export interface VersioningIntentMap {
  UNSPECIFIED: 0;
  COMPATIBLE: 1;
  DEFAULT: 2;
}

export const VersioningIntent: VersioningIntentMap;

export interface ChildWorkflowCancellationTypeMap {
  CHILD_WF_ABANDON: 0;
  CHILD_WF_TRY_CANCEL: 1;
  CHILD_WF_WAIT_CANCELLATION_COMPLETED: 2;
  CHILD_WF_WAIT_CANCELLATION_REQUESTED: 3;
}

export const ChildWorkflowCancellationType: ChildWorkflowCancellationTypeMap;

export interface ActivityCancellationTypeMap {
  TRY_CANCEL: 0;
  WAIT_CANCELLATION_COMPLETED: 1;
  ABANDON: 2;
}

export const ActivityCancellationType: ActivityCancellationTypeMap;

