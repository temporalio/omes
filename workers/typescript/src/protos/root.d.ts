import * as $protobuf from "protobufjs";
import Long = require("long");
/** Namespace temporal. */
export namespace temporal {

    /** Namespace omes. */
    namespace omes {

        /** Namespace kitchen_sink. */
        namespace kitchen_sink {

            /** Properties of a TestInput. */
            interface ITestInput {

                /** TestInput workflowInput */
                workflowInput?: (temporal.omes.kitchen_sink.IWorkflowInput|null);

                /** TestInput clientSequence */
                clientSequence?: (temporal.omes.kitchen_sink.IClientSequence|null);

                /** TestInput withStartAction */
                withStartAction?: (temporal.omes.kitchen_sink.IWithStartClientAction|null);
            }

            /**
             * The input to the test overall. A copy of this constitutes everything that is needed to reproduce
             * the test.
             */
            class TestInput implements ITestInput {

                /**
                 * Constructs a new TestInput.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.ITestInput);

                /** TestInput workflowInput. */
                public workflowInput?: (temporal.omes.kitchen_sink.IWorkflowInput|null);

                /** TestInput clientSequence. */
                public clientSequence?: (temporal.omes.kitchen_sink.IClientSequence|null);

                /** TestInput withStartAction. */
                public withStartAction?: (temporal.omes.kitchen_sink.IWithStartClientAction|null);

                /**
                 * Creates a new TestInput instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns TestInput instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.ITestInput): temporal.omes.kitchen_sink.TestInput;

                /**
                 * Encodes the specified TestInput message. Does not implicitly {@link temporal.omes.kitchen_sink.TestInput.verify|verify} messages.
                 * @param message TestInput message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.ITestInput, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified TestInput message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.TestInput.verify|verify} messages.
                 * @param message TestInput message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.ITestInput, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a TestInput message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns TestInput
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.TestInput;

                /**
                 * Decodes a TestInput message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns TestInput
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.TestInput;

                /**
                 * Creates a TestInput message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns TestInput
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.TestInput;

                /**
                 * Creates a plain object from a TestInput message. Also converts values to other types if specified.
                 * @param message TestInput
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.TestInput, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this TestInput to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for TestInput
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            /** Properties of a ClientSequence. */
            interface IClientSequence {

                /** ClientSequence actionSets */
                actionSets?: (temporal.omes.kitchen_sink.IClientActionSet[]|null);
            }

            /** All the client actions that will be taken over the course of this test */
            class ClientSequence implements IClientSequence {

                /**
                 * Constructs a new ClientSequence.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.IClientSequence);

                /** ClientSequence actionSets. */
                public actionSets: temporal.omes.kitchen_sink.IClientActionSet[];

                /**
                 * Creates a new ClientSequence instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns ClientSequence instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.IClientSequence): temporal.omes.kitchen_sink.ClientSequence;

                /**
                 * Encodes the specified ClientSequence message. Does not implicitly {@link temporal.omes.kitchen_sink.ClientSequence.verify|verify} messages.
                 * @param message ClientSequence message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.IClientSequence, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified ClientSequence message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.ClientSequence.verify|verify} messages.
                 * @param message ClientSequence message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.IClientSequence, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a ClientSequence message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns ClientSequence
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.ClientSequence;

                /**
                 * Decodes a ClientSequence message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns ClientSequence
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.ClientSequence;

                /**
                 * Creates a ClientSequence message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns ClientSequence
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.ClientSequence;

                /**
                 * Creates a plain object from a ClientSequence message. Also converts values to other types if specified.
                 * @param message ClientSequence
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.ClientSequence, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this ClientSequence to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for ClientSequence
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            /** Properties of a ClientActionSet. */
            interface IClientActionSet {

                /** ClientActionSet actions */
                actions?: (temporal.omes.kitchen_sink.IClientAction[]|null);

                /** ClientActionSet concurrent */
                concurrent?: (boolean|null);

                /**
                 * Wait the specified amount of time at the end of the action set before proceeding to the next
                 * (if there is one, if not, ignored).
                 */
                waitAtEnd?: (google.protobuf.IDuration|null);

                /**
                 * If set, the client should wait for the current run to end before proceeding (IE: the workflow
                 * is going to continue-as-new).
                 */
                waitForCurrentRunToFinishAtEnd?: (boolean|null);
            }

            /** A set of client actions to execute. */
            class ClientActionSet implements IClientActionSet {

                /**
                 * Constructs a new ClientActionSet.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.IClientActionSet);

                /** ClientActionSet actions. */
                public actions: temporal.omes.kitchen_sink.IClientAction[];

                /** ClientActionSet concurrent. */
                public concurrent: boolean;

                /**
                 * Wait the specified amount of time at the end of the action set before proceeding to the next
                 * (if there is one, if not, ignored).
                 */
                public waitAtEnd?: (google.protobuf.IDuration|null);

                /**
                 * If set, the client should wait for the current run to end before proceeding (IE: the workflow
                 * is going to continue-as-new).
                 */
                public waitForCurrentRunToFinishAtEnd: boolean;

                /**
                 * Creates a new ClientActionSet instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns ClientActionSet instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.IClientActionSet): temporal.omes.kitchen_sink.ClientActionSet;

                /**
                 * Encodes the specified ClientActionSet message. Does not implicitly {@link temporal.omes.kitchen_sink.ClientActionSet.verify|verify} messages.
                 * @param message ClientActionSet message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.IClientActionSet, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified ClientActionSet message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.ClientActionSet.verify|verify} messages.
                 * @param message ClientActionSet message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.IClientActionSet, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a ClientActionSet message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns ClientActionSet
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.ClientActionSet;

                /**
                 * Decodes a ClientActionSet message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns ClientActionSet
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.ClientActionSet;

                /**
                 * Creates a ClientActionSet message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns ClientActionSet
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.ClientActionSet;

                /**
                 * Creates a plain object from a ClientActionSet message. Also converts values to other types if specified.
                 * @param message ClientActionSet
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.ClientActionSet, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this ClientActionSet to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for ClientActionSet
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            /** Properties of a WithStartClientAction. */
            interface IWithStartClientAction {

                /** WithStartClientAction doSignal */
                doSignal?: (temporal.omes.kitchen_sink.IDoSignal|null);

                /** WithStartClientAction doUpdate */
                doUpdate?: (temporal.omes.kitchen_sink.IDoUpdate|null);
            }

            /** Represents a WithStartClientAction. */
            class WithStartClientAction implements IWithStartClientAction {

                /**
                 * Constructs a new WithStartClientAction.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.IWithStartClientAction);

                /** WithStartClientAction doSignal. */
                public doSignal?: (temporal.omes.kitchen_sink.IDoSignal|null);

                /** WithStartClientAction doUpdate. */
                public doUpdate?: (temporal.omes.kitchen_sink.IDoUpdate|null);

                /** WithStartClientAction variant. */
                public variant?: ("doSignal"|"doUpdate");

                /**
                 * Creates a new WithStartClientAction instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns WithStartClientAction instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.IWithStartClientAction): temporal.omes.kitchen_sink.WithStartClientAction;

                /**
                 * Encodes the specified WithStartClientAction message. Does not implicitly {@link temporal.omes.kitchen_sink.WithStartClientAction.verify|verify} messages.
                 * @param message WithStartClientAction message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.IWithStartClientAction, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified WithStartClientAction message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.WithStartClientAction.verify|verify} messages.
                 * @param message WithStartClientAction message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.IWithStartClientAction, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a WithStartClientAction message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns WithStartClientAction
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.WithStartClientAction;

                /**
                 * Decodes a WithStartClientAction message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns WithStartClientAction
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.WithStartClientAction;

                /**
                 * Creates a WithStartClientAction message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns WithStartClientAction
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.WithStartClientAction;

                /**
                 * Creates a plain object from a WithStartClientAction message. Also converts values to other types if specified.
                 * @param message WithStartClientAction
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.WithStartClientAction, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this WithStartClientAction to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for WithStartClientAction
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            /** Properties of a ClientAction. */
            interface IClientAction {

                /** ClientAction doSignal */
                doSignal?: (temporal.omes.kitchen_sink.IDoSignal|null);

                /** ClientAction doQuery */
                doQuery?: (temporal.omes.kitchen_sink.IDoQuery|null);

                /** ClientAction doUpdate */
                doUpdate?: (temporal.omes.kitchen_sink.IDoUpdate|null);

                /** ClientAction nestedActions */
                nestedActions?: (temporal.omes.kitchen_sink.IClientActionSet|null);

                /** ClientAction doDescribe */
                doDescribe?: (temporal.omes.kitchen_sink.IDoDescribe|null);

                /** ClientAction doStandaloneActivity */
                doStandaloneActivity?: (temporal.omes.kitchen_sink.IDoStandaloneActivity|null);
            }

            /** Represents a ClientAction. */
            class ClientAction implements IClientAction {

                /**
                 * Constructs a new ClientAction.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.IClientAction);

                /** ClientAction doSignal. */
                public doSignal?: (temporal.omes.kitchen_sink.IDoSignal|null);

                /** ClientAction doQuery. */
                public doQuery?: (temporal.omes.kitchen_sink.IDoQuery|null);

                /** ClientAction doUpdate. */
                public doUpdate?: (temporal.omes.kitchen_sink.IDoUpdate|null);

                /** ClientAction nestedActions. */
                public nestedActions?: (temporal.omes.kitchen_sink.IClientActionSet|null);

                /** ClientAction doDescribe. */
                public doDescribe?: (temporal.omes.kitchen_sink.IDoDescribe|null);

                /** ClientAction doStandaloneActivity. */
                public doStandaloneActivity?: (temporal.omes.kitchen_sink.IDoStandaloneActivity|null);

                /** ClientAction variant. */
                public variant?: ("doSignal"|"doQuery"|"doUpdate"|"nestedActions"|"doDescribe"|"doStandaloneActivity");

                /**
                 * Creates a new ClientAction instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns ClientAction instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.IClientAction): temporal.omes.kitchen_sink.ClientAction;

                /**
                 * Encodes the specified ClientAction message. Does not implicitly {@link temporal.omes.kitchen_sink.ClientAction.verify|verify} messages.
                 * @param message ClientAction message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.IClientAction, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified ClientAction message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.ClientAction.verify|verify} messages.
                 * @param message ClientAction message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.IClientAction, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a ClientAction message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns ClientAction
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.ClientAction;

                /**
                 * Decodes a ClientAction message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns ClientAction
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.ClientAction;

                /**
                 * Creates a ClientAction message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns ClientAction
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.ClientAction;

                /**
                 * Creates a plain object from a ClientAction message. Also converts values to other types if specified.
                 * @param message ClientAction
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.ClientAction, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this ClientAction to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for ClientAction
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            /** Properties of a DoSignal. */
            interface IDoSignal {

                /**
                 * A signal handler must exist named `do_actions_signal` which is responsible for handling the
                 * DoSignalActions message. See it's doc for details.
                 */
                doSignalActions?: (temporal.omes.kitchen_sink.DoSignal.IDoSignalActions|null);

                /** Send an arbitrary signal */
                custom?: (temporal.omes.kitchen_sink.IHandlerInvocation|null);

                /** If set, the Signal is a Signal-with-Start. */
                withStart?: (boolean|null);
            }

            /** Represents a DoSignal. */
            class DoSignal implements IDoSignal {

                /**
                 * Constructs a new DoSignal.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.IDoSignal);

                /**
                 * A signal handler must exist named `do_actions_signal` which is responsible for handling the
                 * DoSignalActions message. See it's doc for details.
                 */
                public doSignalActions?: (temporal.omes.kitchen_sink.DoSignal.IDoSignalActions|null);

                /** Send an arbitrary signal */
                public custom?: (temporal.omes.kitchen_sink.IHandlerInvocation|null);

                /** If set, the Signal is a Signal-with-Start. */
                public withStart: boolean;

                /** DoSignal variant. */
                public variant?: ("doSignalActions"|"custom");

                /**
                 * Creates a new DoSignal instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns DoSignal instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.IDoSignal): temporal.omes.kitchen_sink.DoSignal;

                /**
                 * Encodes the specified DoSignal message. Does not implicitly {@link temporal.omes.kitchen_sink.DoSignal.verify|verify} messages.
                 * @param message DoSignal message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.IDoSignal, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified DoSignal message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.DoSignal.verify|verify} messages.
                 * @param message DoSignal message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.IDoSignal, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a DoSignal message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns DoSignal
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.DoSignal;

                /**
                 * Decodes a DoSignal message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns DoSignal
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.DoSignal;

                /**
                 * Creates a DoSignal message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns DoSignal
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.DoSignal;

                /**
                 * Creates a plain object from a DoSignal message. Also converts values to other types if specified.
                 * @param message DoSignal
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.DoSignal, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this DoSignal to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for DoSignal
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            namespace DoSignal {

                /** Properties of a DoSignalActions. */
                interface IDoSignalActions {

                    /**
                     * Execute the action set in the handler. Since Go doesn't have explicit signal handlers it
                     * should instead run the actions in a goroutine for both of these variants, as the
                     * distinction doesn't really matter there.
                     */
                    doActions?: (temporal.omes.kitchen_sink.IActionSet|null);

                    /**
                     * Pipe the actions back to the main workflow function via a queue or similar mechanism, where
                     * they will then be run.
                     */
                    doActionsInMain?: (temporal.omes.kitchen_sink.IActionSet|null);

                    /** The id of the signal to send (1-based indexing). This is used to track the signal and ensure that the worker properly deduplicates signals. */
                    signalId?: (number|null);
                }

                /** Represents a DoSignalActions. */
                class DoSignalActions implements IDoSignalActions {

                    /**
                     * Constructs a new DoSignalActions.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.omes.kitchen_sink.DoSignal.IDoSignalActions);

                    /**
                     * Execute the action set in the handler. Since Go doesn't have explicit signal handlers it
                     * should instead run the actions in a goroutine for both of these variants, as the
                     * distinction doesn't really matter there.
                     */
                    public doActions?: (temporal.omes.kitchen_sink.IActionSet|null);

                    /**
                     * Pipe the actions back to the main workflow function via a queue or similar mechanism, where
                     * they will then be run.
                     */
                    public doActionsInMain?: (temporal.omes.kitchen_sink.IActionSet|null);

                    /** The id of the signal to send (1-based indexing). This is used to track the signal and ensure that the worker properly deduplicates signals. */
                    public signalId: number;

                    /** DoSignalActions variant. */
                    public variant?: ("doActions"|"doActionsInMain");

                    /**
                     * Creates a new DoSignalActions instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns DoSignalActions instance
                     */
                    public static create(properties?: temporal.omes.kitchen_sink.DoSignal.IDoSignalActions): temporal.omes.kitchen_sink.DoSignal.DoSignalActions;

                    /**
                     * Encodes the specified DoSignalActions message. Does not implicitly {@link temporal.omes.kitchen_sink.DoSignal.DoSignalActions.verify|verify} messages.
                     * @param message DoSignalActions message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.omes.kitchen_sink.DoSignal.IDoSignalActions, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified DoSignalActions message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.DoSignal.DoSignalActions.verify|verify} messages.
                     * @param message DoSignalActions message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.omes.kitchen_sink.DoSignal.IDoSignalActions, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a DoSignalActions message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns DoSignalActions
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.DoSignal.DoSignalActions;

                    /**
                     * Decodes a DoSignalActions message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns DoSignalActions
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.DoSignal.DoSignalActions;

                    /**
                     * Creates a DoSignalActions message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns DoSignalActions
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.DoSignal.DoSignalActions;

                    /**
                     * Creates a plain object from a DoSignalActions message. Also converts values to other types if specified.
                     * @param message DoSignalActions
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.omes.kitchen_sink.DoSignal.DoSignalActions, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this DoSignalActions to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for DoSignalActions
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }
            }

            /** Properties of a DoDescribe. */
            interface IDoDescribe {
            }

            /** Represents a DoDescribe. */
            class DoDescribe implements IDoDescribe {

                /**
                 * Constructs a new DoDescribe.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.IDoDescribe);

                /**
                 * Creates a new DoDescribe instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns DoDescribe instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.IDoDescribe): temporal.omes.kitchen_sink.DoDescribe;

                /**
                 * Encodes the specified DoDescribe message. Does not implicitly {@link temporal.omes.kitchen_sink.DoDescribe.verify|verify} messages.
                 * @param message DoDescribe message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.IDoDescribe, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified DoDescribe message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.DoDescribe.verify|verify} messages.
                 * @param message DoDescribe message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.IDoDescribe, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a DoDescribe message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns DoDescribe
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.DoDescribe;

                /**
                 * Decodes a DoDescribe message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns DoDescribe
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.DoDescribe;

                /**
                 * Creates a DoDescribe message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns DoDescribe
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.DoDescribe;

                /**
                 * Creates a plain object from a DoDescribe message. Also converts values to other types if specified.
                 * @param message DoDescribe
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.DoDescribe, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this DoDescribe to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for DoDescribe
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            /** Properties of a DoStandaloneActivity. */
            interface IDoStandaloneActivity {

                /** DoStandaloneActivity activity */
                activity?: (temporal.omes.kitchen_sink.IExecuteActivityAction|null);
            }

            /**
             * Invoke an activity as a standalone activity. Requires server-side support
             * for workflow-independent activities.
             */
            class DoStandaloneActivity implements IDoStandaloneActivity {

                /**
                 * Constructs a new DoStandaloneActivity.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.IDoStandaloneActivity);

                /** DoStandaloneActivity activity. */
                public activity?: (temporal.omes.kitchen_sink.IExecuteActivityAction|null);

                /**
                 * Creates a new DoStandaloneActivity instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns DoStandaloneActivity instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.IDoStandaloneActivity): temporal.omes.kitchen_sink.DoStandaloneActivity;

                /**
                 * Encodes the specified DoStandaloneActivity message. Does not implicitly {@link temporal.omes.kitchen_sink.DoStandaloneActivity.verify|verify} messages.
                 * @param message DoStandaloneActivity message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.IDoStandaloneActivity, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified DoStandaloneActivity message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.DoStandaloneActivity.verify|verify} messages.
                 * @param message DoStandaloneActivity message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.IDoStandaloneActivity, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a DoStandaloneActivity message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns DoStandaloneActivity
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.DoStandaloneActivity;

                /**
                 * Decodes a DoStandaloneActivity message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns DoStandaloneActivity
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.DoStandaloneActivity;

                /**
                 * Creates a DoStandaloneActivity message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns DoStandaloneActivity
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.DoStandaloneActivity;

                /**
                 * Creates a plain object from a DoStandaloneActivity message. Also converts values to other types if specified.
                 * @param message DoStandaloneActivity
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.DoStandaloneActivity, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this DoStandaloneActivity to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for DoStandaloneActivity
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            /** Properties of a DoQuery. */
            interface IDoQuery {

                /**
                 * A query must exist named `report_state` which returns the `WorkflowState` message. The input
                 * is pointless and only exists to allow testing of variably-sized query args.
                 */
                reportState?: (temporal.api.common.v1.IPayloads|null);

                /** Send an arbitrary query */
                custom?: (temporal.omes.kitchen_sink.IHandlerInvocation|null);

                /** If set, the client should expect the query to fail */
                failureExpected?: (boolean|null);
            }

            /** Represents a DoQuery. */
            class DoQuery implements IDoQuery {

                /**
                 * Constructs a new DoQuery.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.IDoQuery);

                /**
                 * A query must exist named `report_state` which returns the `WorkflowState` message. The input
                 * is pointless and only exists to allow testing of variably-sized query args.
                 */
                public reportState?: (temporal.api.common.v1.IPayloads|null);

                /** Send an arbitrary query */
                public custom?: (temporal.omes.kitchen_sink.IHandlerInvocation|null);

                /** If set, the client should expect the query to fail */
                public failureExpected: boolean;

                /** DoQuery variant. */
                public variant?: ("reportState"|"custom");

                /**
                 * Creates a new DoQuery instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns DoQuery instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.IDoQuery): temporal.omes.kitchen_sink.DoQuery;

                /**
                 * Encodes the specified DoQuery message. Does not implicitly {@link temporal.omes.kitchen_sink.DoQuery.verify|verify} messages.
                 * @param message DoQuery message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.IDoQuery, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified DoQuery message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.DoQuery.verify|verify} messages.
                 * @param message DoQuery message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.IDoQuery, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a DoQuery message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns DoQuery
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.DoQuery;

                /**
                 * Decodes a DoQuery message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns DoQuery
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.DoQuery;

                /**
                 * Creates a DoQuery message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns DoQuery
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.DoQuery;

                /**
                 * Creates a plain object from a DoQuery message. Also converts values to other types if specified.
                 * @param message DoQuery
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.DoQuery, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this DoQuery to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for DoQuery
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            /** Properties of a DoUpdate. */
            interface IDoUpdate {

                /**
                 * An update must exist named `do_actions_update` which handles the `DoActionsUpdate` message.
                 * See message doc for what it should do.
                 */
                doActions?: (temporal.omes.kitchen_sink.IDoActionsUpdate|null);

                /** Send an arbitrary update request */
                custom?: (temporal.omes.kitchen_sink.IHandlerInvocation|null);

                /** If set, the Update is an Update-with-Start. */
                withStart?: (boolean|null);

                /** If set, the client should expect the update to fail */
                failureExpected?: (boolean|null);
            }

            /** Represents a DoUpdate. */
            class DoUpdate implements IDoUpdate {

                /**
                 * Constructs a new DoUpdate.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.IDoUpdate);

                /**
                 * An update must exist named `do_actions_update` which handles the `DoActionsUpdate` message.
                 * See message doc for what it should do.
                 */
                public doActions?: (temporal.omes.kitchen_sink.IDoActionsUpdate|null);

                /** Send an arbitrary update request */
                public custom?: (temporal.omes.kitchen_sink.IHandlerInvocation|null);

                /** If set, the Update is an Update-with-Start. */
                public withStart: boolean;

                /** If set, the client should expect the update to fail */
                public failureExpected: boolean;

                /** DoUpdate variant. */
                public variant?: ("doActions"|"custom");

                /**
                 * Creates a new DoUpdate instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns DoUpdate instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.IDoUpdate): temporal.omes.kitchen_sink.DoUpdate;

                /**
                 * Encodes the specified DoUpdate message. Does not implicitly {@link temporal.omes.kitchen_sink.DoUpdate.verify|verify} messages.
                 * @param message DoUpdate message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.IDoUpdate, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified DoUpdate message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.DoUpdate.verify|verify} messages.
                 * @param message DoUpdate message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.IDoUpdate, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a DoUpdate message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns DoUpdate
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.DoUpdate;

                /**
                 * Decodes a DoUpdate message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns DoUpdate
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.DoUpdate;

                /**
                 * Creates a DoUpdate message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns DoUpdate
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.DoUpdate;

                /**
                 * Creates a plain object from a DoUpdate message. Also converts values to other types if specified.
                 * @param message DoUpdate
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.DoUpdate, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this DoUpdate to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for DoUpdate
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            /** Properties of a DoActionsUpdate. */
            interface IDoActionsUpdate {

                /**
                 * Do same thing signal handler would do when given the provided action set. The handler should
                 * return the `WorkflowState` when done with all the provided actions. You may also include a
                 * `ReturnErrorAction` or `ContinueAsNewAction` in the set to exit the handler in those ways.
                 */
                doActions?: (temporal.omes.kitchen_sink.IActionSet|null);

                /** The validator should reject the update */
                rejectMe?: (google.protobuf.IEmpty|null);
            }

            /** Represents a DoActionsUpdate. */
            class DoActionsUpdate implements IDoActionsUpdate {

                /**
                 * Constructs a new DoActionsUpdate.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.IDoActionsUpdate);

                /**
                 * Do same thing signal handler would do when given the provided action set. The handler should
                 * return the `WorkflowState` when done with all the provided actions. You may also include a
                 * `ReturnErrorAction` or `ContinueAsNewAction` in the set to exit the handler in those ways.
                 */
                public doActions?: (temporal.omes.kitchen_sink.IActionSet|null);

                /** The validator should reject the update */
                public rejectMe?: (google.protobuf.IEmpty|null);

                /** DoActionsUpdate variant. */
                public variant?: ("doActions"|"rejectMe");

                /**
                 * Creates a new DoActionsUpdate instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns DoActionsUpdate instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.IDoActionsUpdate): temporal.omes.kitchen_sink.DoActionsUpdate;

                /**
                 * Encodes the specified DoActionsUpdate message. Does not implicitly {@link temporal.omes.kitchen_sink.DoActionsUpdate.verify|verify} messages.
                 * @param message DoActionsUpdate message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.IDoActionsUpdate, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified DoActionsUpdate message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.DoActionsUpdate.verify|verify} messages.
                 * @param message DoActionsUpdate message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.IDoActionsUpdate, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a DoActionsUpdate message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns DoActionsUpdate
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.DoActionsUpdate;

                /**
                 * Decodes a DoActionsUpdate message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns DoActionsUpdate
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.DoActionsUpdate;

                /**
                 * Creates a DoActionsUpdate message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns DoActionsUpdate
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.DoActionsUpdate;

                /**
                 * Creates a plain object from a DoActionsUpdate message. Also converts values to other types if specified.
                 * @param message DoActionsUpdate
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.DoActionsUpdate, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this DoActionsUpdate to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for DoActionsUpdate
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            /** Properties of a HandlerInvocation. */
            interface IHandlerInvocation {

                /** HandlerInvocation name */
                name?: (string|null);

                /** HandlerInvocation args */
                args?: (temporal.api.common.v1.IPayload[]|null);
            }

            /** Represents a HandlerInvocation. */
            class HandlerInvocation implements IHandlerInvocation {

                /**
                 * Constructs a new HandlerInvocation.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.IHandlerInvocation);

                /** HandlerInvocation name. */
                public name: string;

                /** HandlerInvocation args. */
                public args: temporal.api.common.v1.IPayload[];

                /**
                 * Creates a new HandlerInvocation instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns HandlerInvocation instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.IHandlerInvocation): temporal.omes.kitchen_sink.HandlerInvocation;

                /**
                 * Encodes the specified HandlerInvocation message. Does not implicitly {@link temporal.omes.kitchen_sink.HandlerInvocation.verify|verify} messages.
                 * @param message HandlerInvocation message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.IHandlerInvocation, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified HandlerInvocation message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.HandlerInvocation.verify|verify} messages.
                 * @param message HandlerInvocation message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.IHandlerInvocation, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a HandlerInvocation message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns HandlerInvocation
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.HandlerInvocation;

                /**
                 * Decodes a HandlerInvocation message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns HandlerInvocation
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.HandlerInvocation;

                /**
                 * Creates a HandlerInvocation message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns HandlerInvocation
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.HandlerInvocation;

                /**
                 * Creates a plain object from a HandlerInvocation message. Also converts values to other types if specified.
                 * @param message HandlerInvocation
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.HandlerInvocation, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this HandlerInvocation to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for HandlerInvocation
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            /** Properties of a WorkflowState. */
            interface IWorkflowState {

                /** WorkflowState kvs */
                kvs?: ({ [k: string]: string }|null);
            }

            /** Each workflow must maintain an instance of this state */
            class WorkflowState implements IWorkflowState {

                /**
                 * Constructs a new WorkflowState.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.IWorkflowState);

                /** WorkflowState kvs. */
                public kvs: { [k: string]: string };

                /**
                 * Creates a new WorkflowState instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns WorkflowState instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.IWorkflowState): temporal.omes.kitchen_sink.WorkflowState;

                /**
                 * Encodes the specified WorkflowState message. Does not implicitly {@link temporal.omes.kitchen_sink.WorkflowState.verify|verify} messages.
                 * @param message WorkflowState message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.IWorkflowState, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified WorkflowState message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.WorkflowState.verify|verify} messages.
                 * @param message WorkflowState message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.IWorkflowState, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a WorkflowState message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns WorkflowState
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.WorkflowState;

                /**
                 * Decodes a WorkflowState message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns WorkflowState
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.WorkflowState;

                /**
                 * Creates a WorkflowState message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns WorkflowState
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.WorkflowState;

                /**
                 * Creates a plain object from a WorkflowState message. Also converts values to other types if specified.
                 * @param message WorkflowState
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.WorkflowState, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this WorkflowState to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for WorkflowState
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            /** Properties of a WorkflowInput. */
            interface IWorkflowInput {

                /** WorkflowInput initialActions */
                initialActions?: (temporal.omes.kitchen_sink.IActionSet[]|null);

                /** Number of signals the client will send to the workflow */
                expectedSignalCount?: (number|null);

                /** Signal de-duplication state (used when continuing as new). Signal IDs use 1-based indexing. */
                expectedSignalIds?: (number[]|null);

                /** WorkflowInput receivedSignalIds */
                receivedSignalIds?: (number[]|null);
            }

            /** Represents a WorkflowInput. */
            class WorkflowInput implements IWorkflowInput {

                /**
                 * Constructs a new WorkflowInput.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.IWorkflowInput);

                /** WorkflowInput initialActions. */
                public initialActions: temporal.omes.kitchen_sink.IActionSet[];

                /** Number of signals the client will send to the workflow */
                public expectedSignalCount: number;

                /** Signal de-duplication state (used when continuing as new). Signal IDs use 1-based indexing. */
                public expectedSignalIds: number[];

                /** WorkflowInput receivedSignalIds. */
                public receivedSignalIds: number[];

                /**
                 * Creates a new WorkflowInput instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns WorkflowInput instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.IWorkflowInput): temporal.omes.kitchen_sink.WorkflowInput;

                /**
                 * Encodes the specified WorkflowInput message. Does not implicitly {@link temporal.omes.kitchen_sink.WorkflowInput.verify|verify} messages.
                 * @param message WorkflowInput message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.IWorkflowInput, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified WorkflowInput message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.WorkflowInput.verify|verify} messages.
                 * @param message WorkflowInput message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.IWorkflowInput, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a WorkflowInput message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns WorkflowInput
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.WorkflowInput;

                /**
                 * Decodes a WorkflowInput message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns WorkflowInput
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.WorkflowInput;

                /**
                 * Creates a WorkflowInput message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns WorkflowInput
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.WorkflowInput;

                /**
                 * Creates a plain object from a WorkflowInput message. Also converts values to other types if specified.
                 * @param message WorkflowInput
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.WorkflowInput, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this WorkflowInput to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for WorkflowInput
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            /** Properties of an ActionSet. */
            interface IActionSet {

                /** ActionSet actions */
                actions?: (temporal.omes.kitchen_sink.IAction[]|null);

                /** ActionSet concurrent */
                concurrent?: (boolean|null);
            }

            /**
             * A set of actions to execute concurrently or sequentially. It is necessary to be able to represent
             * sequential execution without multiple 1-size action sets, as that implies the receipt of a signal
             * between each of those sets, which may not be desired.
             *
             * All actions are handled before proceeding to the next action set, unless one of those actions
             * would cause the workflow to complete/fail/CAN.
             */
            class ActionSet implements IActionSet {

                /**
                 * Constructs a new ActionSet.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.IActionSet);

                /** ActionSet actions. */
                public actions: temporal.omes.kitchen_sink.IAction[];

                /** ActionSet concurrent. */
                public concurrent: boolean;

                /**
                 * Creates a new ActionSet instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns ActionSet instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.IActionSet): temporal.omes.kitchen_sink.ActionSet;

                /**
                 * Encodes the specified ActionSet message. Does not implicitly {@link temporal.omes.kitchen_sink.ActionSet.verify|verify} messages.
                 * @param message ActionSet message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.IActionSet, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified ActionSet message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.ActionSet.verify|verify} messages.
                 * @param message ActionSet message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.IActionSet, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes an ActionSet message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns ActionSet
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.ActionSet;

                /**
                 * Decodes an ActionSet message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns ActionSet
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.ActionSet;

                /**
                 * Creates an ActionSet message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns ActionSet
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.ActionSet;

                /**
                 * Creates a plain object from an ActionSet message. Also converts values to other types if specified.
                 * @param message ActionSet
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.ActionSet, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this ActionSet to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for ActionSet
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            /** Properties of an Action. */
            interface IAction {

                /** Action timer */
                timer?: (temporal.omes.kitchen_sink.ITimerAction|null);

                /** Action execActivity */
                execActivity?: (temporal.omes.kitchen_sink.IExecuteActivityAction|null);

                /** Action execChildWorkflow */
                execChildWorkflow?: (temporal.omes.kitchen_sink.IExecuteChildWorkflowAction|null);

                /** Action awaitWorkflowState */
                awaitWorkflowState?: (temporal.omes.kitchen_sink.IAwaitWorkflowState|null);

                /** Action sendSignal */
                sendSignal?: (temporal.omes.kitchen_sink.ISendSignalAction|null);

                /** Action cancelWorkflow */
                cancelWorkflow?: (temporal.omes.kitchen_sink.ICancelWorkflowAction|null);

                /** Action setPatchMarker */
                setPatchMarker?: (temporal.omes.kitchen_sink.ISetPatchMarkerAction|null);

                /** Action upsertSearchAttributes */
                upsertSearchAttributes?: (temporal.omes.kitchen_sink.IUpsertSearchAttributesAction|null);

                /** Action upsertMemo */
                upsertMemo?: (temporal.omes.kitchen_sink.IUpsertMemoAction|null);

                /** Action setWorkflowState */
                setWorkflowState?: (temporal.omes.kitchen_sink.IWorkflowState|null);

                /** Action returnResult */
                returnResult?: (temporal.omes.kitchen_sink.IReturnResultAction|null);

                /** Action returnError */
                returnError?: (temporal.omes.kitchen_sink.IReturnErrorAction|null);

                /** Action continueAsNew */
                continueAsNew?: (temporal.omes.kitchen_sink.IContinueAsNewAction|null);

                /** Action nestedActionSet */
                nestedActionSet?: (temporal.omes.kitchen_sink.IActionSet|null);

                /** Action nexusOperation */
                nexusOperation?: (temporal.omes.kitchen_sink.IExecuteNexusOperation|null);
            }

            /** Represents an Action. */
            class Action implements IAction {

                /**
                 * Constructs a new Action.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.IAction);

                /** Action timer. */
                public timer?: (temporal.omes.kitchen_sink.ITimerAction|null);

                /** Action execActivity. */
                public execActivity?: (temporal.omes.kitchen_sink.IExecuteActivityAction|null);

                /** Action execChildWorkflow. */
                public execChildWorkflow?: (temporal.omes.kitchen_sink.IExecuteChildWorkflowAction|null);

                /** Action awaitWorkflowState. */
                public awaitWorkflowState?: (temporal.omes.kitchen_sink.IAwaitWorkflowState|null);

                /** Action sendSignal. */
                public sendSignal?: (temporal.omes.kitchen_sink.ISendSignalAction|null);

                /** Action cancelWorkflow. */
                public cancelWorkflow?: (temporal.omes.kitchen_sink.ICancelWorkflowAction|null);

                /** Action setPatchMarker. */
                public setPatchMarker?: (temporal.omes.kitchen_sink.ISetPatchMarkerAction|null);

                /** Action upsertSearchAttributes. */
                public upsertSearchAttributes?: (temporal.omes.kitchen_sink.IUpsertSearchAttributesAction|null);

                /** Action upsertMemo. */
                public upsertMemo?: (temporal.omes.kitchen_sink.IUpsertMemoAction|null);

                /** Action setWorkflowState. */
                public setWorkflowState?: (temporal.omes.kitchen_sink.IWorkflowState|null);

                /** Action returnResult. */
                public returnResult?: (temporal.omes.kitchen_sink.IReturnResultAction|null);

                /** Action returnError. */
                public returnError?: (temporal.omes.kitchen_sink.IReturnErrorAction|null);

                /** Action continueAsNew. */
                public continueAsNew?: (temporal.omes.kitchen_sink.IContinueAsNewAction|null);

                /** Action nestedActionSet. */
                public nestedActionSet?: (temporal.omes.kitchen_sink.IActionSet|null);

                /** Action nexusOperation. */
                public nexusOperation?: (temporal.omes.kitchen_sink.IExecuteNexusOperation|null);

                /** Action variant. */
                public variant?: ("timer"|"execActivity"|"execChildWorkflow"|"awaitWorkflowState"|"sendSignal"|"cancelWorkflow"|"setPatchMarker"|"upsertSearchAttributes"|"upsertMemo"|"setWorkflowState"|"returnResult"|"returnError"|"continueAsNew"|"nestedActionSet"|"nexusOperation");

                /**
                 * Creates a new Action instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns Action instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.IAction): temporal.omes.kitchen_sink.Action;

                /**
                 * Encodes the specified Action message. Does not implicitly {@link temporal.omes.kitchen_sink.Action.verify|verify} messages.
                 * @param message Action message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.IAction, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified Action message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.Action.verify|verify} messages.
                 * @param message Action message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.IAction, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes an Action message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns Action
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.Action;

                /**
                 * Decodes an Action message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns Action
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.Action;

                /**
                 * Creates an Action message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns Action
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.Action;

                /**
                 * Creates a plain object from an Action message. Also converts values to other types if specified.
                 * @param message Action
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.Action, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this Action to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for Action
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            /** Properties of an AwaitableChoice. */
            interface IAwaitableChoice {

                /** Wait for the command to finish. This is the default. */
                waitFinish?: (google.protobuf.IEmpty|null);

                /** Call the command API but immediately move on. IE: Don't await on it at all. */
                abandon?: (google.protobuf.IEmpty|null);

                /**
                 * Cancel the command before it's begun - IE: Cancel it immediately after starting it with
                 * no await, within the same workflow task.
                 */
                cancelBeforeStarted?: (google.protobuf.IEmpty|null);

                /**
                 * Cancel the command after it's been started. Not all SDKs will know when a command is started
                 * and in those cases they should issue the cancellation in the next workflow task after
                 * creating the command.
                 */
                cancelAfterStarted?: (google.protobuf.IEmpty|null);

                /** Cancel the command after it's already completed. */
                cancelAfterCompleted?: (google.protobuf.IEmpty|null);
            }

            /**
             * All await commands will have this available as a field. If it is set, the command
             * should be either awaited upon, cancelled, or abandoned at the specified juncture (if possible,
             * not all command types will be cancellable at all stages. Is is up to the generator to produce
             * valid conditions).
             */
            class AwaitableChoice implements IAwaitableChoice {

                /**
                 * Constructs a new AwaitableChoice.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.IAwaitableChoice);

                /** Wait for the command to finish. This is the default. */
                public waitFinish?: (google.protobuf.IEmpty|null);

                /** Call the command API but immediately move on. IE: Don't await on it at all. */
                public abandon?: (google.protobuf.IEmpty|null);

                /**
                 * Cancel the command before it's begun - IE: Cancel it immediately after starting it with
                 * no await, within the same workflow task.
                 */
                public cancelBeforeStarted?: (google.protobuf.IEmpty|null);

                /**
                 * Cancel the command after it's been started. Not all SDKs will know when a command is started
                 * and in those cases they should issue the cancellation in the next workflow task after
                 * creating the command.
                 */
                public cancelAfterStarted?: (google.protobuf.IEmpty|null);

                /** Cancel the command after it's already completed. */
                public cancelAfterCompleted?: (google.protobuf.IEmpty|null);

                /** AwaitableChoice condition. */
                public condition?: ("waitFinish"|"abandon"|"cancelBeforeStarted"|"cancelAfterStarted"|"cancelAfterCompleted");

                /**
                 * Creates a new AwaitableChoice instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns AwaitableChoice instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.IAwaitableChoice): temporal.omes.kitchen_sink.AwaitableChoice;

                /**
                 * Encodes the specified AwaitableChoice message. Does not implicitly {@link temporal.omes.kitchen_sink.AwaitableChoice.verify|verify} messages.
                 * @param message AwaitableChoice message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.IAwaitableChoice, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified AwaitableChoice message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.AwaitableChoice.verify|verify} messages.
                 * @param message AwaitableChoice message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.IAwaitableChoice, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes an AwaitableChoice message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns AwaitableChoice
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.AwaitableChoice;

                /**
                 * Decodes an AwaitableChoice message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns AwaitableChoice
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.AwaitableChoice;

                /**
                 * Creates an AwaitableChoice message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns AwaitableChoice
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.AwaitableChoice;

                /**
                 * Creates a plain object from an AwaitableChoice message. Also converts values to other types if specified.
                 * @param message AwaitableChoice
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.AwaitableChoice, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this AwaitableChoice to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for AwaitableChoice
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            /** Properties of a TimerAction. */
            interface ITimerAction {

                /** TimerAction milliseconds */
                milliseconds?: (Long|null);

                /** TimerAction awaitableChoice */
                awaitableChoice?: (temporal.omes.kitchen_sink.IAwaitableChoice|null);
            }

            /** Represents a TimerAction. */
            class TimerAction implements ITimerAction {

                /**
                 * Constructs a new TimerAction.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.ITimerAction);

                /** TimerAction milliseconds. */
                public milliseconds: Long;

                /** TimerAction awaitableChoice. */
                public awaitableChoice?: (temporal.omes.kitchen_sink.IAwaitableChoice|null);

                /**
                 * Creates a new TimerAction instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns TimerAction instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.ITimerAction): temporal.omes.kitchen_sink.TimerAction;

                /**
                 * Encodes the specified TimerAction message. Does not implicitly {@link temporal.omes.kitchen_sink.TimerAction.verify|verify} messages.
                 * @param message TimerAction message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.ITimerAction, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified TimerAction message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.TimerAction.verify|verify} messages.
                 * @param message TimerAction message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.ITimerAction, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a TimerAction message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns TimerAction
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.TimerAction;

                /**
                 * Decodes a TimerAction message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns TimerAction
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.TimerAction;

                /**
                 * Creates a TimerAction message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns TimerAction
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.TimerAction;

                /**
                 * Creates a plain object from a TimerAction message. Also converts values to other types if specified.
                 * @param message TimerAction
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.TimerAction, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this TimerAction to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for TimerAction
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            /** Properties of an ExecuteActivityAction. */
            interface IExecuteActivityAction {

                /** ExecuteActivityAction generic */
                generic?: (temporal.omes.kitchen_sink.ExecuteActivityAction.IGenericActivity|null);

                /**
                 * There must be an activity named `delay` which accepts some kind of duration and waits
                 * for that long
                 */
                delay?: (google.protobuf.IDuration|null);

                /** There must be an activity named `noop` which does nothing */
                noop?: (google.protobuf.IEmpty|null);

                /** There must be an activity named `resources` which accepts the ResourcesActivity message as input */
                resources?: (temporal.omes.kitchen_sink.ExecuteActivityAction.IResourcesActivity|null);

                /** There must be an activity named `payload` which accepts the PayloadActivity message as input */
                payload?: (temporal.omes.kitchen_sink.ExecuteActivityAction.IPayloadActivity|null);

                /** There must be an activity named `client` which accepts the ClientActivity message as input */
                client?: (temporal.omes.kitchen_sink.ExecuteActivityAction.IClientActivity|null);

                /** There must be an activity named `retryable_error` which accepts the RetryableErrorActivity message as input */
                retryableError?: (temporal.omes.kitchen_sink.ExecuteActivityAction.IRetryableErrorActivity|null);

                /** There must be an activity named `timeout` which accepts the TimeoutActivity message as input */
                timeout?: (temporal.omes.kitchen_sink.ExecuteActivityAction.ITimeoutActivity|null);

                /** There must be an activity named `heartbeat` which accepts the HeartbeatTimeoutActivity message as input */
                heartbeat?: (temporal.omes.kitchen_sink.ExecuteActivityAction.IHeartbeatTimeoutActivity|null);

                /** The name of the task queue to place this activity request in */
                taskQueue?: (string|null);

                /** ExecuteActivityAction headers */
                headers?: ({ [k: string]: temporal.api.common.v1.IPayload }|null);

                /**
                 * Indicates how long the caller is willing to wait for an activity completion. Limits how long
                 * retries will be attempted. Either this or start_to_close_timeout_seconds must be specified.
                 * When not specified defaults to the workflow execution timeout.
                 */
                scheduleToCloseTimeout?: (google.protobuf.IDuration|null);

                /**
                 * Limits time an activity task can stay in a task queue before a worker picks it up. This
                 * timeout is always non retryable as all a retry would achieve is to put it back into the same
                 * queue. Defaults to schedule_to_close_timeout or workflow execution timeout if not specified.
                 */
                scheduleToStartTimeout?: (google.protobuf.IDuration|null);

                /**
                 * Maximum time an activity is allowed to execute after a pick up by a worker. This timeout is
                 * always retryable. Either this or schedule_to_close_timeout must be specified.
                 */
                startToCloseTimeout?: (google.protobuf.IDuration|null);

                /** Maximum time allowed between successful worker heartbeats. */
                heartbeatTimeout?: (google.protobuf.IDuration|null);

                /**
                 * Activities are provided by a default retry policy controlled through the service dynamic
                 * configuration. Retries are happening up to schedule_to_close_timeout. To disable retries set
                 * retry_policy.maximum_attempts to 1.
                 */
                retryPolicy?: (temporal.api.common.v1.IRetryPolicy|null);

                /** ExecuteActivityAction isLocal */
                isLocal?: (google.protobuf.IEmpty|null);

                /** ExecuteActivityAction remote */
                remote?: (temporal.omes.kitchen_sink.IRemoteActivityOptions|null);

                /** ExecuteActivityAction awaitableChoice */
                awaitableChoice?: (temporal.omes.kitchen_sink.IAwaitableChoice|null);

                /** ExecuteActivityAction priority */
                priority?: (temporal.api.common.v1.IPriority|null);

                /** TODO: once complete, use commonpb.PriorityKey instead */
                fairnessKey?: (string|null);

                /** ExecuteActivityAction fairnessWeight */
                fairnessWeight?: (number|null);
            }

            /** Represents an ExecuteActivityAction. */
            class ExecuteActivityAction implements IExecuteActivityAction {

                /**
                 * Constructs a new ExecuteActivityAction.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.IExecuteActivityAction);

                /** ExecuteActivityAction generic. */
                public generic?: (temporal.omes.kitchen_sink.ExecuteActivityAction.IGenericActivity|null);

                /**
                 * There must be an activity named `delay` which accepts some kind of duration and waits
                 * for that long
                 */
                public delay?: (google.protobuf.IDuration|null);

                /** There must be an activity named `noop` which does nothing */
                public noop?: (google.protobuf.IEmpty|null);

                /** There must be an activity named `resources` which accepts the ResourcesActivity message as input */
                public resources?: (temporal.omes.kitchen_sink.ExecuteActivityAction.IResourcesActivity|null);

                /** There must be an activity named `payload` which accepts the PayloadActivity message as input */
                public payload?: (temporal.omes.kitchen_sink.ExecuteActivityAction.IPayloadActivity|null);

                /** There must be an activity named `client` which accepts the ClientActivity message as input */
                public client?: (temporal.omes.kitchen_sink.ExecuteActivityAction.IClientActivity|null);

                /** There must be an activity named `retryable_error` which accepts the RetryableErrorActivity message as input */
                public retryableError?: (temporal.omes.kitchen_sink.ExecuteActivityAction.IRetryableErrorActivity|null);

                /** There must be an activity named `timeout` which accepts the TimeoutActivity message as input */
                public timeout?: (temporal.omes.kitchen_sink.ExecuteActivityAction.ITimeoutActivity|null);

                /** There must be an activity named `heartbeat` which accepts the HeartbeatTimeoutActivity message as input */
                public heartbeat?: (temporal.omes.kitchen_sink.ExecuteActivityAction.IHeartbeatTimeoutActivity|null);

                /** The name of the task queue to place this activity request in */
                public taskQueue: string;

                /** ExecuteActivityAction headers. */
                public headers: { [k: string]: temporal.api.common.v1.IPayload };

                /**
                 * Indicates how long the caller is willing to wait for an activity completion. Limits how long
                 * retries will be attempted. Either this or start_to_close_timeout_seconds must be specified.
                 * When not specified defaults to the workflow execution timeout.
                 */
                public scheduleToCloseTimeout?: (google.protobuf.IDuration|null);

                /**
                 * Limits time an activity task can stay in a task queue before a worker picks it up. This
                 * timeout is always non retryable as all a retry would achieve is to put it back into the same
                 * queue. Defaults to schedule_to_close_timeout or workflow execution timeout if not specified.
                 */
                public scheduleToStartTimeout?: (google.protobuf.IDuration|null);

                /**
                 * Maximum time an activity is allowed to execute after a pick up by a worker. This timeout is
                 * always retryable. Either this or schedule_to_close_timeout must be specified.
                 */
                public startToCloseTimeout?: (google.protobuf.IDuration|null);

                /** Maximum time allowed between successful worker heartbeats. */
                public heartbeatTimeout?: (google.protobuf.IDuration|null);

                /**
                 * Activities are provided by a default retry policy controlled through the service dynamic
                 * configuration. Retries are happening up to schedule_to_close_timeout. To disable retries set
                 * retry_policy.maximum_attempts to 1.
                 */
                public retryPolicy?: (temporal.api.common.v1.IRetryPolicy|null);

                /** ExecuteActivityAction isLocal. */
                public isLocal?: (google.protobuf.IEmpty|null);

                /** ExecuteActivityAction remote. */
                public remote?: (temporal.omes.kitchen_sink.IRemoteActivityOptions|null);

                /** ExecuteActivityAction awaitableChoice. */
                public awaitableChoice?: (temporal.omes.kitchen_sink.IAwaitableChoice|null);

                /** ExecuteActivityAction priority. */
                public priority?: (temporal.api.common.v1.IPriority|null);

                /** TODO: once complete, use commonpb.PriorityKey instead */
                public fairnessKey: string;

                /** ExecuteActivityAction fairnessWeight. */
                public fairnessWeight: number;

                /** ExecuteActivityAction activityType. */
                public activityType?: ("generic"|"delay"|"noop"|"resources"|"payload"|"client"|"retryableError"|"timeout"|"heartbeat");

                /** Whether or not this activity will be a local activity */
                public locality?: ("isLocal"|"remote");

                /**
                 * Creates a new ExecuteActivityAction instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns ExecuteActivityAction instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.IExecuteActivityAction): temporal.omes.kitchen_sink.ExecuteActivityAction;

                /**
                 * Encodes the specified ExecuteActivityAction message. Does not implicitly {@link temporal.omes.kitchen_sink.ExecuteActivityAction.verify|verify} messages.
                 * @param message ExecuteActivityAction message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.IExecuteActivityAction, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified ExecuteActivityAction message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.ExecuteActivityAction.verify|verify} messages.
                 * @param message ExecuteActivityAction message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.IExecuteActivityAction, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes an ExecuteActivityAction message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns ExecuteActivityAction
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.ExecuteActivityAction;

                /**
                 * Decodes an ExecuteActivityAction message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns ExecuteActivityAction
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.ExecuteActivityAction;

                /**
                 * Creates an ExecuteActivityAction message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns ExecuteActivityAction
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.ExecuteActivityAction;

                /**
                 * Creates a plain object from an ExecuteActivityAction message. Also converts values to other types if specified.
                 * @param message ExecuteActivityAction
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.ExecuteActivityAction, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this ExecuteActivityAction to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for ExecuteActivityAction
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            namespace ExecuteActivityAction {

                /** Properties of a GenericActivity. */
                interface IGenericActivity {

                    /** GenericActivity type */
                    type?: (string|null);

                    /** GenericActivity arguments */
                    "arguments"?: (temporal.api.common.v1.IPayload[]|null);
                }

                /** Represents a GenericActivity. */
                class GenericActivity implements IGenericActivity {

                    /**
                     * Constructs a new GenericActivity.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.omes.kitchen_sink.ExecuteActivityAction.IGenericActivity);

                    /** GenericActivity type. */
                    public type: string;

                    /** GenericActivity arguments. */
                    public arguments: temporal.api.common.v1.IPayload[];

                    /**
                     * Creates a new GenericActivity instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns GenericActivity instance
                     */
                    public static create(properties?: temporal.omes.kitchen_sink.ExecuteActivityAction.IGenericActivity): temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity;

                    /**
                     * Encodes the specified GenericActivity message. Does not implicitly {@link temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity.verify|verify} messages.
                     * @param message GenericActivity message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.omes.kitchen_sink.ExecuteActivityAction.IGenericActivity, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified GenericActivity message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity.verify|verify} messages.
                     * @param message GenericActivity message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.omes.kitchen_sink.ExecuteActivityAction.IGenericActivity, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a GenericActivity message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns GenericActivity
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity;

                    /**
                     * Decodes a GenericActivity message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns GenericActivity
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity;

                    /**
                     * Creates a GenericActivity message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns GenericActivity
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity;

                    /**
                     * Creates a plain object from a GenericActivity message. Also converts values to other types if specified.
                     * @param message GenericActivity
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.omes.kitchen_sink.ExecuteActivityAction.GenericActivity, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this GenericActivity to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for GenericActivity
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of a ResourcesActivity. */
                interface IResourcesActivity {

                    /** ResourcesActivity runFor */
                    runFor?: (google.protobuf.IDuration|null);

                    /** ResourcesActivity bytesToAllocate */
                    bytesToAllocate?: (Long|null);

                    /** ResourcesActivity cpuYieldEveryNIterations */
                    cpuYieldEveryNIterations?: (number|null);

                    /** ResourcesActivity cpuYieldForMs */
                    cpuYieldForMs?: (number|null);
                }

                /** Represents a ResourcesActivity. */
                class ResourcesActivity implements IResourcesActivity {

                    /**
                     * Constructs a new ResourcesActivity.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.omes.kitchen_sink.ExecuteActivityAction.IResourcesActivity);

                    /** ResourcesActivity runFor. */
                    public runFor?: (google.protobuf.IDuration|null);

                    /** ResourcesActivity bytesToAllocate. */
                    public bytesToAllocate: Long;

                    /** ResourcesActivity cpuYieldEveryNIterations. */
                    public cpuYieldEveryNIterations: number;

                    /** ResourcesActivity cpuYieldForMs. */
                    public cpuYieldForMs: number;

                    /**
                     * Creates a new ResourcesActivity instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns ResourcesActivity instance
                     */
                    public static create(properties?: temporal.omes.kitchen_sink.ExecuteActivityAction.IResourcesActivity): temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity;

                    /**
                     * Encodes the specified ResourcesActivity message. Does not implicitly {@link temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity.verify|verify} messages.
                     * @param message ResourcesActivity message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.omes.kitchen_sink.ExecuteActivityAction.IResourcesActivity, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified ResourcesActivity message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity.verify|verify} messages.
                     * @param message ResourcesActivity message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.omes.kitchen_sink.ExecuteActivityAction.IResourcesActivity, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a ResourcesActivity message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns ResourcesActivity
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity;

                    /**
                     * Decodes a ResourcesActivity message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns ResourcesActivity
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity;

                    /**
                     * Creates a ResourcesActivity message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns ResourcesActivity
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity;

                    /**
                     * Creates a plain object from a ResourcesActivity message. Also converts values to other types if specified.
                     * @param message ResourcesActivity
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this ResourcesActivity to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for ResourcesActivity
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of a PayloadActivity. */
                interface IPayloadActivity {

                    /** PayloadActivity bytesToReceive */
                    bytesToReceive?: (number|null);

                    /** PayloadActivity bytesToReturn */
                    bytesToReturn?: (number|null);
                }

                /** Represents a PayloadActivity. */
                class PayloadActivity implements IPayloadActivity {

                    /**
                     * Constructs a new PayloadActivity.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.omes.kitchen_sink.ExecuteActivityAction.IPayloadActivity);

                    /** PayloadActivity bytesToReceive. */
                    public bytesToReceive: number;

                    /** PayloadActivity bytesToReturn. */
                    public bytesToReturn: number;

                    /**
                     * Creates a new PayloadActivity instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns PayloadActivity instance
                     */
                    public static create(properties?: temporal.omes.kitchen_sink.ExecuteActivityAction.IPayloadActivity): temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity;

                    /**
                     * Encodes the specified PayloadActivity message. Does not implicitly {@link temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity.verify|verify} messages.
                     * @param message PayloadActivity message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.omes.kitchen_sink.ExecuteActivityAction.IPayloadActivity, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified PayloadActivity message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity.verify|verify} messages.
                     * @param message PayloadActivity message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.omes.kitchen_sink.ExecuteActivityAction.IPayloadActivity, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a PayloadActivity message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns PayloadActivity
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity;

                    /**
                     * Decodes a PayloadActivity message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns PayloadActivity
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity;

                    /**
                     * Creates a PayloadActivity message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns PayloadActivity
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity;

                    /**
                     * Creates a plain object from a PayloadActivity message. Also converts values to other types if specified.
                     * @param message PayloadActivity
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.omes.kitchen_sink.ExecuteActivityAction.PayloadActivity, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this PayloadActivity to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for PayloadActivity
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of a ClientActivity. */
                interface IClientActivity {

                    /** ClientActivity clientSequence */
                    clientSequence?: (temporal.omes.kitchen_sink.IClientSequence|null);
                }

                /** Represents a ClientActivity. */
                class ClientActivity implements IClientActivity {

                    /**
                     * Constructs a new ClientActivity.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.omes.kitchen_sink.ExecuteActivityAction.IClientActivity);

                    /** ClientActivity clientSequence. */
                    public clientSequence?: (temporal.omes.kitchen_sink.IClientSequence|null);

                    /**
                     * Creates a new ClientActivity instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns ClientActivity instance
                     */
                    public static create(properties?: temporal.omes.kitchen_sink.ExecuteActivityAction.IClientActivity): temporal.omes.kitchen_sink.ExecuteActivityAction.ClientActivity;

                    /**
                     * Encodes the specified ClientActivity message. Does not implicitly {@link temporal.omes.kitchen_sink.ExecuteActivityAction.ClientActivity.verify|verify} messages.
                     * @param message ClientActivity message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.omes.kitchen_sink.ExecuteActivityAction.IClientActivity, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified ClientActivity message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.ExecuteActivityAction.ClientActivity.verify|verify} messages.
                     * @param message ClientActivity message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.omes.kitchen_sink.ExecuteActivityAction.IClientActivity, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a ClientActivity message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns ClientActivity
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.ExecuteActivityAction.ClientActivity;

                    /**
                     * Decodes a ClientActivity message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns ClientActivity
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.ExecuteActivityAction.ClientActivity;

                    /**
                     * Creates a ClientActivity message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns ClientActivity
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.ExecuteActivityAction.ClientActivity;

                    /**
                     * Creates a plain object from a ClientActivity message. Also converts values to other types if specified.
                     * @param message ClientActivity
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.omes.kitchen_sink.ExecuteActivityAction.ClientActivity, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this ClientActivity to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for ClientActivity
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of a RetryableErrorActivity. */
                interface IRetryableErrorActivity {

                    /** How many attempts should fail before succeeding (1-indexed) */
                    failAttempts?: (number|null);
                }

                /**
                 * Activity that throws retryable errors for N attempts, then succeeds.
                 * Tests activity retry behavior.
                 */
                class RetryableErrorActivity implements IRetryableErrorActivity {

                    /**
                     * Constructs a new RetryableErrorActivity.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.omes.kitchen_sink.ExecuteActivityAction.IRetryableErrorActivity);

                    /** How many attempts should fail before succeeding (1-indexed) */
                    public failAttempts: number;

                    /**
                     * Creates a new RetryableErrorActivity instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns RetryableErrorActivity instance
                     */
                    public static create(properties?: temporal.omes.kitchen_sink.ExecuteActivityAction.IRetryableErrorActivity): temporal.omes.kitchen_sink.ExecuteActivityAction.RetryableErrorActivity;

                    /**
                     * Encodes the specified RetryableErrorActivity message. Does not implicitly {@link temporal.omes.kitchen_sink.ExecuteActivityAction.RetryableErrorActivity.verify|verify} messages.
                     * @param message RetryableErrorActivity message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.omes.kitchen_sink.ExecuteActivityAction.IRetryableErrorActivity, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified RetryableErrorActivity message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.ExecuteActivityAction.RetryableErrorActivity.verify|verify} messages.
                     * @param message RetryableErrorActivity message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.omes.kitchen_sink.ExecuteActivityAction.IRetryableErrorActivity, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a RetryableErrorActivity message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns RetryableErrorActivity
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.ExecuteActivityAction.RetryableErrorActivity;

                    /**
                     * Decodes a RetryableErrorActivity message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns RetryableErrorActivity
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.ExecuteActivityAction.RetryableErrorActivity;

                    /**
                     * Creates a RetryableErrorActivity message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns RetryableErrorActivity
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.ExecuteActivityAction.RetryableErrorActivity;

                    /**
                     * Creates a plain object from a RetryableErrorActivity message. Also converts values to other types if specified.
                     * @param message RetryableErrorActivity
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.omes.kitchen_sink.ExecuteActivityAction.RetryableErrorActivity, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this RetryableErrorActivity to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for RetryableErrorActivity
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of a TimeoutActivity. */
                interface ITimeoutActivity {

                    /** How many attempts should timeout before succeeding (1-indexed) */
                    failAttempts?: (number|null);

                    /** Duration to run on success, must be less than StartToClose timeout */
                    successDuration?: (google.protobuf.IDuration|null);

                    /** Duration to run on failure, must be more than StartToClose timeout */
                    failureDuration?: (google.protobuf.IDuration|null);
                }

                /**
                 * Activity that runs too long for N attempts (causing timeout), then completes.
                 * Tests StartToCloseTimeout behavior with retries.
                 */
                class TimeoutActivity implements ITimeoutActivity {

                    /**
                     * Constructs a new TimeoutActivity.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.omes.kitchen_sink.ExecuteActivityAction.ITimeoutActivity);

                    /** How many attempts should timeout before succeeding (1-indexed) */
                    public failAttempts: number;

                    /** Duration to run on success, must be less than StartToClose timeout */
                    public successDuration?: (google.protobuf.IDuration|null);

                    /** Duration to run on failure, must be more than StartToClose timeout */
                    public failureDuration?: (google.protobuf.IDuration|null);

                    /**
                     * Creates a new TimeoutActivity instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns TimeoutActivity instance
                     */
                    public static create(properties?: temporal.omes.kitchen_sink.ExecuteActivityAction.ITimeoutActivity): temporal.omes.kitchen_sink.ExecuteActivityAction.TimeoutActivity;

                    /**
                     * Encodes the specified TimeoutActivity message. Does not implicitly {@link temporal.omes.kitchen_sink.ExecuteActivityAction.TimeoutActivity.verify|verify} messages.
                     * @param message TimeoutActivity message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.omes.kitchen_sink.ExecuteActivityAction.ITimeoutActivity, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified TimeoutActivity message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.ExecuteActivityAction.TimeoutActivity.verify|verify} messages.
                     * @param message TimeoutActivity message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.omes.kitchen_sink.ExecuteActivityAction.ITimeoutActivity, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a TimeoutActivity message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns TimeoutActivity
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.ExecuteActivityAction.TimeoutActivity;

                    /**
                     * Decodes a TimeoutActivity message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns TimeoutActivity
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.ExecuteActivityAction.TimeoutActivity;

                    /**
                     * Creates a TimeoutActivity message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns TimeoutActivity
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.ExecuteActivityAction.TimeoutActivity;

                    /**
                     * Creates a plain object from a TimeoutActivity message. Also converts values to other types if specified.
                     * @param message TimeoutActivity
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.omes.kitchen_sink.ExecuteActivityAction.TimeoutActivity, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this TimeoutActivity to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for TimeoutActivity
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of a HeartbeatTimeoutActivity. */
                interface IHeartbeatTimeoutActivity {

                    /** How many attempts should skip heartbeats before succeeding (1-indexed) */
                    failAttempts?: (number|null);

                    /** Duration to run on success, must be less than HeartbeatTimeout timeout */
                    successDuration?: (google.protobuf.IDuration|null);

                    /** Duration to run on failure, must be more than HeartbeatTimeout timeout */
                    failureDuration?: (google.protobuf.IDuration|null);
                }

                /**
                 * Activity that skips heartbeats for N attempts (causing heartbeat timeout), then sends them.
                 * Tests HeartbeatTimeout behavior with retries.
                 */
                class HeartbeatTimeoutActivity implements IHeartbeatTimeoutActivity {

                    /**
                     * Constructs a new HeartbeatTimeoutActivity.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.omes.kitchen_sink.ExecuteActivityAction.IHeartbeatTimeoutActivity);

                    /** How many attempts should skip heartbeats before succeeding (1-indexed) */
                    public failAttempts: number;

                    /** Duration to run on success, must be less than HeartbeatTimeout timeout */
                    public successDuration?: (google.protobuf.IDuration|null);

                    /** Duration to run on failure, must be more than HeartbeatTimeout timeout */
                    public failureDuration?: (google.protobuf.IDuration|null);

                    /**
                     * Creates a new HeartbeatTimeoutActivity instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns HeartbeatTimeoutActivity instance
                     */
                    public static create(properties?: temporal.omes.kitchen_sink.ExecuteActivityAction.IHeartbeatTimeoutActivity): temporal.omes.kitchen_sink.ExecuteActivityAction.HeartbeatTimeoutActivity;

                    /**
                     * Encodes the specified HeartbeatTimeoutActivity message. Does not implicitly {@link temporal.omes.kitchen_sink.ExecuteActivityAction.HeartbeatTimeoutActivity.verify|verify} messages.
                     * @param message HeartbeatTimeoutActivity message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.omes.kitchen_sink.ExecuteActivityAction.IHeartbeatTimeoutActivity, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified HeartbeatTimeoutActivity message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.ExecuteActivityAction.HeartbeatTimeoutActivity.verify|verify} messages.
                     * @param message HeartbeatTimeoutActivity message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.omes.kitchen_sink.ExecuteActivityAction.IHeartbeatTimeoutActivity, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a HeartbeatTimeoutActivity message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns HeartbeatTimeoutActivity
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.ExecuteActivityAction.HeartbeatTimeoutActivity;

                    /**
                     * Decodes a HeartbeatTimeoutActivity message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns HeartbeatTimeoutActivity
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.ExecuteActivityAction.HeartbeatTimeoutActivity;

                    /**
                     * Creates a HeartbeatTimeoutActivity message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns HeartbeatTimeoutActivity
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.ExecuteActivityAction.HeartbeatTimeoutActivity;

                    /**
                     * Creates a plain object from a HeartbeatTimeoutActivity message. Also converts values to other types if specified.
                     * @param message HeartbeatTimeoutActivity
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.omes.kitchen_sink.ExecuteActivityAction.HeartbeatTimeoutActivity, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this HeartbeatTimeoutActivity to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for HeartbeatTimeoutActivity
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }
            }

            /** Properties of an ExecuteChildWorkflowAction. */
            interface IExecuteChildWorkflowAction {

                /** ExecuteChildWorkflowAction namespace */
                namespace?: (string|null);

                /** ExecuteChildWorkflowAction workflowId */
                workflowId?: (string|null);

                /** ExecuteChildWorkflowAction workflowType */
                workflowType?: (string|null);

                /** ExecuteChildWorkflowAction taskQueue */
                taskQueue?: (string|null);

                /** ExecuteChildWorkflowAction input */
                input?: (temporal.api.common.v1.IPayload[]|null);

                /** Total workflow execution timeout including retries and continue as new. */
                workflowExecutionTimeout?: (google.protobuf.IDuration|null);

                /** Timeout of a single workflow run. */
                workflowRunTimeout?: (google.protobuf.IDuration|null);

                /** Timeout of a single workflow task. */
                workflowTaskTimeout?: (google.protobuf.IDuration|null);

                /** Default: PARENT_CLOSE_POLICY_TERMINATE. */
                parentClosePolicy?: (temporal.omes.kitchen_sink.ParentClosePolicy|null);

                /** Default: WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE. */
                workflowIdReusePolicy?: (temporal.api.enums.v1.WorkflowIdReusePolicy|null);

                /** ExecuteChildWorkflowAction retryPolicy */
                retryPolicy?: (temporal.api.common.v1.IRetryPolicy|null);

                /** ExecuteChildWorkflowAction cronSchedule */
                cronSchedule?: (string|null);

                /** Header fields */
                headers?: ({ [k: string]: temporal.api.common.v1.IPayload }|null);

                /** Memo fields */
                memo?: ({ [k: string]: temporal.api.common.v1.IPayload }|null);

                /** Search attributes */
                searchAttributes?: ({ [k: string]: temporal.api.common.v1.IPayload }|null);

                /** Defines behaviour of the underlying workflow when child workflow cancellation has been requested. */
                cancellationType?: (temporal.omes.kitchen_sink.ChildWorkflowCancellationType|null);

                /** Whether this child should run on a worker with a compatible build id or not. */
                versioningIntent?: (temporal.omes.kitchen_sink.VersioningIntent|null);

                /** ExecuteChildWorkflowAction awaitableChoice */
                awaitableChoice?: (temporal.omes.kitchen_sink.IAwaitableChoice|null);
            }

            /** Represents an ExecuteChildWorkflowAction. */
            class ExecuteChildWorkflowAction implements IExecuteChildWorkflowAction {

                /**
                 * Constructs a new ExecuteChildWorkflowAction.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.IExecuteChildWorkflowAction);

                /** ExecuteChildWorkflowAction namespace. */
                public namespace: string;

                /** ExecuteChildWorkflowAction workflowId. */
                public workflowId: string;

                /** ExecuteChildWorkflowAction workflowType. */
                public workflowType: string;

                /** ExecuteChildWorkflowAction taskQueue. */
                public taskQueue: string;

                /** ExecuteChildWorkflowAction input. */
                public input: temporal.api.common.v1.IPayload[];

                /** Total workflow execution timeout including retries and continue as new. */
                public workflowExecutionTimeout?: (google.protobuf.IDuration|null);

                /** Timeout of a single workflow run. */
                public workflowRunTimeout?: (google.protobuf.IDuration|null);

                /** Timeout of a single workflow task. */
                public workflowTaskTimeout?: (google.protobuf.IDuration|null);

                /** Default: PARENT_CLOSE_POLICY_TERMINATE. */
                public parentClosePolicy: temporal.omes.kitchen_sink.ParentClosePolicy;

                /** Default: WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE. */
                public workflowIdReusePolicy: temporal.api.enums.v1.WorkflowIdReusePolicy;

                /** ExecuteChildWorkflowAction retryPolicy. */
                public retryPolicy?: (temporal.api.common.v1.IRetryPolicy|null);

                /** ExecuteChildWorkflowAction cronSchedule. */
                public cronSchedule: string;

                /** Header fields */
                public headers: { [k: string]: temporal.api.common.v1.IPayload };

                /** Memo fields */
                public memo: { [k: string]: temporal.api.common.v1.IPayload };

                /** Search attributes */
                public searchAttributes: { [k: string]: temporal.api.common.v1.IPayload };

                /** Defines behaviour of the underlying workflow when child workflow cancellation has been requested. */
                public cancellationType: temporal.omes.kitchen_sink.ChildWorkflowCancellationType;

                /** Whether this child should run on a worker with a compatible build id or not. */
                public versioningIntent: temporal.omes.kitchen_sink.VersioningIntent;

                /** ExecuteChildWorkflowAction awaitableChoice. */
                public awaitableChoice?: (temporal.omes.kitchen_sink.IAwaitableChoice|null);

                /**
                 * Creates a new ExecuteChildWorkflowAction instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns ExecuteChildWorkflowAction instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.IExecuteChildWorkflowAction): temporal.omes.kitchen_sink.ExecuteChildWorkflowAction;

                /**
                 * Encodes the specified ExecuteChildWorkflowAction message. Does not implicitly {@link temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.verify|verify} messages.
                 * @param message ExecuteChildWorkflowAction message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.IExecuteChildWorkflowAction, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified ExecuteChildWorkflowAction message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.ExecuteChildWorkflowAction.verify|verify} messages.
                 * @param message ExecuteChildWorkflowAction message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.IExecuteChildWorkflowAction, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes an ExecuteChildWorkflowAction message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns ExecuteChildWorkflowAction
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.ExecuteChildWorkflowAction;

                /**
                 * Decodes an ExecuteChildWorkflowAction message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns ExecuteChildWorkflowAction
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.ExecuteChildWorkflowAction;

                /**
                 * Creates an ExecuteChildWorkflowAction message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns ExecuteChildWorkflowAction
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.ExecuteChildWorkflowAction;

                /**
                 * Creates a plain object from an ExecuteChildWorkflowAction message. Also converts values to other types if specified.
                 * @param message ExecuteChildWorkflowAction
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.ExecuteChildWorkflowAction, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this ExecuteChildWorkflowAction to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for ExecuteChildWorkflowAction
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            /** Properties of an AwaitWorkflowState. */
            interface IAwaitWorkflowState {

                /** AwaitWorkflowState key */
                key?: (string|null);

                /** AwaitWorkflowState value */
                value?: (string|null);
            }

            /** Wait for the workflow state to have a matching k/v entry */
            class AwaitWorkflowState implements IAwaitWorkflowState {

                /**
                 * Constructs a new AwaitWorkflowState.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.IAwaitWorkflowState);

                /** AwaitWorkflowState key. */
                public key: string;

                /** AwaitWorkflowState value. */
                public value: string;

                /**
                 * Creates a new AwaitWorkflowState instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns AwaitWorkflowState instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.IAwaitWorkflowState): temporal.omes.kitchen_sink.AwaitWorkflowState;

                /**
                 * Encodes the specified AwaitWorkflowState message. Does not implicitly {@link temporal.omes.kitchen_sink.AwaitWorkflowState.verify|verify} messages.
                 * @param message AwaitWorkflowState message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.IAwaitWorkflowState, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified AwaitWorkflowState message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.AwaitWorkflowState.verify|verify} messages.
                 * @param message AwaitWorkflowState message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.IAwaitWorkflowState, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes an AwaitWorkflowState message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns AwaitWorkflowState
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.AwaitWorkflowState;

                /**
                 * Decodes an AwaitWorkflowState message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns AwaitWorkflowState
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.AwaitWorkflowState;

                /**
                 * Creates an AwaitWorkflowState message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns AwaitWorkflowState
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.AwaitWorkflowState;

                /**
                 * Creates a plain object from an AwaitWorkflowState message. Also converts values to other types if specified.
                 * @param message AwaitWorkflowState
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.AwaitWorkflowState, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this AwaitWorkflowState to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for AwaitWorkflowState
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            /** Properties of a SendSignalAction. */
            interface ISendSignalAction {

                /** What workflow is being targeted */
                workflowId?: (string|null);

                /** SendSignalAction runId */
                runId?: (string|null);

                /** Name of the signal handler */
                signalName?: (string|null);

                /** Arguments for the handler */
                args?: (temporal.api.common.v1.IPayload[]|null);

                /** Headers to attach to the signal */
                headers?: ({ [k: string]: temporal.api.common.v1.IPayload }|null);

                /** SendSignalAction awaitableChoice */
                awaitableChoice?: (temporal.omes.kitchen_sink.IAwaitableChoice|null);
            }

            /** Represents a SendSignalAction. */
            class SendSignalAction implements ISendSignalAction {

                /**
                 * Constructs a new SendSignalAction.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.ISendSignalAction);

                /** What workflow is being targeted */
                public workflowId: string;

                /** SendSignalAction runId. */
                public runId: string;

                /** Name of the signal handler */
                public signalName: string;

                /** Arguments for the handler */
                public args: temporal.api.common.v1.IPayload[];

                /** Headers to attach to the signal */
                public headers: { [k: string]: temporal.api.common.v1.IPayload };

                /** SendSignalAction awaitableChoice. */
                public awaitableChoice?: (temporal.omes.kitchen_sink.IAwaitableChoice|null);

                /**
                 * Creates a new SendSignalAction instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns SendSignalAction instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.ISendSignalAction): temporal.omes.kitchen_sink.SendSignalAction;

                /**
                 * Encodes the specified SendSignalAction message. Does not implicitly {@link temporal.omes.kitchen_sink.SendSignalAction.verify|verify} messages.
                 * @param message SendSignalAction message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.ISendSignalAction, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified SendSignalAction message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.SendSignalAction.verify|verify} messages.
                 * @param message SendSignalAction message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.ISendSignalAction, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a SendSignalAction message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns SendSignalAction
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.SendSignalAction;

                /**
                 * Decodes a SendSignalAction message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns SendSignalAction
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.SendSignalAction;

                /**
                 * Creates a SendSignalAction message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns SendSignalAction
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.SendSignalAction;

                /**
                 * Creates a plain object from a SendSignalAction message. Also converts values to other types if specified.
                 * @param message SendSignalAction
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.SendSignalAction, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this SendSignalAction to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for SendSignalAction
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            /** Properties of a CancelWorkflowAction. */
            interface ICancelWorkflowAction {

                /** CancelWorkflowAction workflowId */
                workflowId?: (string|null);

                /** CancelWorkflowAction runId */
                runId?: (string|null);
            }

            /** Cancel an external workflow (may be a child) */
            class CancelWorkflowAction implements ICancelWorkflowAction {

                /**
                 * Constructs a new CancelWorkflowAction.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.ICancelWorkflowAction);

                /** CancelWorkflowAction workflowId. */
                public workflowId: string;

                /** CancelWorkflowAction runId. */
                public runId: string;

                /**
                 * Creates a new CancelWorkflowAction instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns CancelWorkflowAction instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.ICancelWorkflowAction): temporal.omes.kitchen_sink.CancelWorkflowAction;

                /**
                 * Encodes the specified CancelWorkflowAction message. Does not implicitly {@link temporal.omes.kitchen_sink.CancelWorkflowAction.verify|verify} messages.
                 * @param message CancelWorkflowAction message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.ICancelWorkflowAction, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified CancelWorkflowAction message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.CancelWorkflowAction.verify|verify} messages.
                 * @param message CancelWorkflowAction message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.ICancelWorkflowAction, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a CancelWorkflowAction message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns CancelWorkflowAction
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.CancelWorkflowAction;

                /**
                 * Decodes a CancelWorkflowAction message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns CancelWorkflowAction
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.CancelWorkflowAction;

                /**
                 * Creates a CancelWorkflowAction message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns CancelWorkflowAction
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.CancelWorkflowAction;

                /**
                 * Creates a plain object from a CancelWorkflowAction message. Also converts values to other types if specified.
                 * @param message CancelWorkflowAction
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.CancelWorkflowAction, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this CancelWorkflowAction to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for CancelWorkflowAction
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            /** Properties of a SetPatchMarkerAction. */
            interface ISetPatchMarkerAction {

                /**
                 * A user-chosen identifier for this patch. If the same identifier is used in multiple places in
                 * the code, those places are considered to be versioned as one unit. IE: The check call will
                 * return the same result for all of them
                 */
                patchId?: (string|null);

                /**
                 * TODO Not sure how we could use this in these tests
                 * Can be set to true to indicate that branches using this change are being removed, and all
                 * future worker deployments will only have the "with change" code in them.
                 */
                deprecated?: (boolean|null);

                /** Perform this action behind the if guard */
                innerAction?: (temporal.omes.kitchen_sink.IAction|null);
            }

            /**
             * patched or getVersion API
             * For getVersion SDKs, use `DEFAULT_VERSION, 1` as the numeric arguments,
             */
            class SetPatchMarkerAction implements ISetPatchMarkerAction {

                /**
                 * Constructs a new SetPatchMarkerAction.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.ISetPatchMarkerAction);

                /**
                 * A user-chosen identifier for this patch. If the same identifier is used in multiple places in
                 * the code, those places are considered to be versioned as one unit. IE: The check call will
                 * return the same result for all of them
                 */
                public patchId: string;

                /**
                 * TODO Not sure how we could use this in these tests
                 * Can be set to true to indicate that branches using this change are being removed, and all
                 * future worker deployments will only have the "with change" code in them.
                 */
                public deprecated: boolean;

                /** Perform this action behind the if guard */
                public innerAction?: (temporal.omes.kitchen_sink.IAction|null);

                /**
                 * Creates a new SetPatchMarkerAction instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns SetPatchMarkerAction instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.ISetPatchMarkerAction): temporal.omes.kitchen_sink.SetPatchMarkerAction;

                /**
                 * Encodes the specified SetPatchMarkerAction message. Does not implicitly {@link temporal.omes.kitchen_sink.SetPatchMarkerAction.verify|verify} messages.
                 * @param message SetPatchMarkerAction message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.ISetPatchMarkerAction, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified SetPatchMarkerAction message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.SetPatchMarkerAction.verify|verify} messages.
                 * @param message SetPatchMarkerAction message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.ISetPatchMarkerAction, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a SetPatchMarkerAction message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns SetPatchMarkerAction
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.SetPatchMarkerAction;

                /**
                 * Decodes a SetPatchMarkerAction message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns SetPatchMarkerAction
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.SetPatchMarkerAction;

                /**
                 * Creates a SetPatchMarkerAction message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns SetPatchMarkerAction
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.SetPatchMarkerAction;

                /**
                 * Creates a plain object from a SetPatchMarkerAction message. Also converts values to other types if specified.
                 * @param message SetPatchMarkerAction
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.SetPatchMarkerAction, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this SetPatchMarkerAction to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for SetPatchMarkerAction
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            /** Properties of an UpsertSearchAttributesAction. */
            interface IUpsertSearchAttributesAction {

                /**
                 * SearchAttributes fields - equivalent to indexed_fields on api. Key = search index, Value =
                 * value
                 */
                searchAttributes?: ({ [k: string]: temporal.api.common.v1.IPayload }|null);
            }

            /** Represents an UpsertSearchAttributesAction. */
            class UpsertSearchAttributesAction implements IUpsertSearchAttributesAction {

                /**
                 * Constructs a new UpsertSearchAttributesAction.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.IUpsertSearchAttributesAction);

                /**
                 * SearchAttributes fields - equivalent to indexed_fields on api. Key = search index, Value =
                 * value
                 */
                public searchAttributes: { [k: string]: temporal.api.common.v1.IPayload };

                /**
                 * Creates a new UpsertSearchAttributesAction instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns UpsertSearchAttributesAction instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.IUpsertSearchAttributesAction): temporal.omes.kitchen_sink.UpsertSearchAttributesAction;

                /**
                 * Encodes the specified UpsertSearchAttributesAction message. Does not implicitly {@link temporal.omes.kitchen_sink.UpsertSearchAttributesAction.verify|verify} messages.
                 * @param message UpsertSearchAttributesAction message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.IUpsertSearchAttributesAction, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified UpsertSearchAttributesAction message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.UpsertSearchAttributesAction.verify|verify} messages.
                 * @param message UpsertSearchAttributesAction message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.IUpsertSearchAttributesAction, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes an UpsertSearchAttributesAction message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns UpsertSearchAttributesAction
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.UpsertSearchAttributesAction;

                /**
                 * Decodes an UpsertSearchAttributesAction message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns UpsertSearchAttributesAction
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.UpsertSearchAttributesAction;

                /**
                 * Creates an UpsertSearchAttributesAction message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns UpsertSearchAttributesAction
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.UpsertSearchAttributesAction;

                /**
                 * Creates a plain object from an UpsertSearchAttributesAction message. Also converts values to other types if specified.
                 * @param message UpsertSearchAttributesAction
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.UpsertSearchAttributesAction, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this UpsertSearchAttributesAction to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for UpsertSearchAttributesAction
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            /** Properties of an UpsertMemoAction. */
            interface IUpsertMemoAction {

                /**
                 * Update the workflow memo with the provided values. The values will be merged with
                 * the existing memo. If the user wants to delete values, a default/empty Payload should be
                 * used as the value for the key being deleted.
                 */
                upsertedMemo?: (temporal.api.common.v1.IMemo|null);
            }

            /** Represents an UpsertMemoAction. */
            class UpsertMemoAction implements IUpsertMemoAction {

                /**
                 * Constructs a new UpsertMemoAction.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.IUpsertMemoAction);

                /**
                 * Update the workflow memo with the provided values. The values will be merged with
                 * the existing memo. If the user wants to delete values, a default/empty Payload should be
                 * used as the value for the key being deleted.
                 */
                public upsertedMemo?: (temporal.api.common.v1.IMemo|null);

                /**
                 * Creates a new UpsertMemoAction instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns UpsertMemoAction instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.IUpsertMemoAction): temporal.omes.kitchen_sink.UpsertMemoAction;

                /**
                 * Encodes the specified UpsertMemoAction message. Does not implicitly {@link temporal.omes.kitchen_sink.UpsertMemoAction.verify|verify} messages.
                 * @param message UpsertMemoAction message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.IUpsertMemoAction, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified UpsertMemoAction message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.UpsertMemoAction.verify|verify} messages.
                 * @param message UpsertMemoAction message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.IUpsertMemoAction, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes an UpsertMemoAction message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns UpsertMemoAction
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.UpsertMemoAction;

                /**
                 * Decodes an UpsertMemoAction message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns UpsertMemoAction
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.UpsertMemoAction;

                /**
                 * Creates an UpsertMemoAction message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns UpsertMemoAction
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.UpsertMemoAction;

                /**
                 * Creates a plain object from an UpsertMemoAction message. Also converts values to other types if specified.
                 * @param message UpsertMemoAction
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.UpsertMemoAction, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this UpsertMemoAction to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for UpsertMemoAction
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            /** Properties of a ReturnResultAction. */
            interface IReturnResultAction {

                /** ReturnResultAction returnThis */
                returnThis?: (temporal.api.common.v1.IPayload|null);
            }

            /** Represents a ReturnResultAction. */
            class ReturnResultAction implements IReturnResultAction {

                /**
                 * Constructs a new ReturnResultAction.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.IReturnResultAction);

                /** ReturnResultAction returnThis. */
                public returnThis?: (temporal.api.common.v1.IPayload|null);

                /**
                 * Creates a new ReturnResultAction instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns ReturnResultAction instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.IReturnResultAction): temporal.omes.kitchen_sink.ReturnResultAction;

                /**
                 * Encodes the specified ReturnResultAction message. Does not implicitly {@link temporal.omes.kitchen_sink.ReturnResultAction.verify|verify} messages.
                 * @param message ReturnResultAction message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.IReturnResultAction, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified ReturnResultAction message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.ReturnResultAction.verify|verify} messages.
                 * @param message ReturnResultAction message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.IReturnResultAction, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a ReturnResultAction message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns ReturnResultAction
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.ReturnResultAction;

                /**
                 * Decodes a ReturnResultAction message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns ReturnResultAction
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.ReturnResultAction;

                /**
                 * Creates a ReturnResultAction message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns ReturnResultAction
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.ReturnResultAction;

                /**
                 * Creates a plain object from a ReturnResultAction message. Also converts values to other types if specified.
                 * @param message ReturnResultAction
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.ReturnResultAction, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this ReturnResultAction to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for ReturnResultAction
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            /** Properties of a ReturnErrorAction. */
            interface IReturnErrorAction {

                /** ReturnErrorAction failure */
                failure?: (temporal.api.failure.v1.IFailure|null);
            }

            /** Represents a ReturnErrorAction. */
            class ReturnErrorAction implements IReturnErrorAction {

                /**
                 * Constructs a new ReturnErrorAction.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.IReturnErrorAction);

                /** ReturnErrorAction failure. */
                public failure?: (temporal.api.failure.v1.IFailure|null);

                /**
                 * Creates a new ReturnErrorAction instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns ReturnErrorAction instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.IReturnErrorAction): temporal.omes.kitchen_sink.ReturnErrorAction;

                /**
                 * Encodes the specified ReturnErrorAction message. Does not implicitly {@link temporal.omes.kitchen_sink.ReturnErrorAction.verify|verify} messages.
                 * @param message ReturnErrorAction message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.IReturnErrorAction, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified ReturnErrorAction message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.ReturnErrorAction.verify|verify} messages.
                 * @param message ReturnErrorAction message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.IReturnErrorAction, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a ReturnErrorAction message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns ReturnErrorAction
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.ReturnErrorAction;

                /**
                 * Decodes a ReturnErrorAction message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns ReturnErrorAction
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.ReturnErrorAction;

                /**
                 * Creates a ReturnErrorAction message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns ReturnErrorAction
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.ReturnErrorAction;

                /**
                 * Creates a plain object from a ReturnErrorAction message. Also converts values to other types if specified.
                 * @param message ReturnErrorAction
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.ReturnErrorAction, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this ReturnErrorAction to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for ReturnErrorAction
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            /** Properties of a ContinueAsNewAction. */
            interface IContinueAsNewAction {

                /** The identifier the lang-specific sdk uses to execute workflow code */
                workflowType?: (string|null);

                /** Task queue for the new workflow execution */
                taskQueue?: (string|null);

                /**
                 * Inputs to the workflow code. Should be specified. Will not re-use old arguments, as that
                 * typically wouldn't make any sense.
                 */
                "arguments"?: (temporal.api.common.v1.IPayload[]|null);

                /** Timeout for a single run of the new workflow. Will not re-use current workflow's value. */
                workflowRunTimeout?: (google.protobuf.IDuration|null);

                /** Timeout of a single workflow task. Will not re-use current workflow's value. */
                workflowTaskTimeout?: (google.protobuf.IDuration|null);

                /** If set, the new workflow will have this memo. If unset, re-uses the current workflow's memo */
                memo?: ({ [k: string]: temporal.api.common.v1.IPayload }|null);

                /**
                 * If set, the new workflow will have these headers. Will *not* re-use current workflow's
                 * headers otherwise.
                 */
                headers?: ({ [k: string]: temporal.api.common.v1.IPayload }|null);

                /**
                 * If set, the new workflow will have these search attributes. If unset, re-uses the current
                 * workflow's search attributes.
                 */
                searchAttributes?: ({ [k: string]: temporal.api.common.v1.IPayload }|null);

                /**
                 * If set, the new workflow will have this retry policy. If unset, re-uses the current
                 * workflow's retry policy.
                 */
                retryPolicy?: (temporal.api.common.v1.IRetryPolicy|null);

                /** Whether the continued workflow should run on a worker with a compatible build id or not. */
                versioningIntent?: (temporal.omes.kitchen_sink.VersioningIntent|null);
            }

            /** Represents a ContinueAsNewAction. */
            class ContinueAsNewAction implements IContinueAsNewAction {

                /**
                 * Constructs a new ContinueAsNewAction.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.IContinueAsNewAction);

                /** The identifier the lang-specific sdk uses to execute workflow code */
                public workflowType: string;

                /** Task queue for the new workflow execution */
                public taskQueue: string;

                /**
                 * Inputs to the workflow code. Should be specified. Will not re-use old arguments, as that
                 * typically wouldn't make any sense.
                 */
                public arguments: temporal.api.common.v1.IPayload[];

                /** Timeout for a single run of the new workflow. Will not re-use current workflow's value. */
                public workflowRunTimeout?: (google.protobuf.IDuration|null);

                /** Timeout of a single workflow task. Will not re-use current workflow's value. */
                public workflowTaskTimeout?: (google.protobuf.IDuration|null);

                /** If set, the new workflow will have this memo. If unset, re-uses the current workflow's memo */
                public memo: { [k: string]: temporal.api.common.v1.IPayload };

                /**
                 * If set, the new workflow will have these headers. Will *not* re-use current workflow's
                 * headers otherwise.
                 */
                public headers: { [k: string]: temporal.api.common.v1.IPayload };

                /**
                 * If set, the new workflow will have these search attributes. If unset, re-uses the current
                 * workflow's search attributes.
                 */
                public searchAttributes: { [k: string]: temporal.api.common.v1.IPayload };

                /**
                 * If set, the new workflow will have this retry policy. If unset, re-uses the current
                 * workflow's retry policy.
                 */
                public retryPolicy?: (temporal.api.common.v1.IRetryPolicy|null);

                /** Whether the continued workflow should run on a worker with a compatible build id or not. */
                public versioningIntent: temporal.omes.kitchen_sink.VersioningIntent;

                /**
                 * Creates a new ContinueAsNewAction instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns ContinueAsNewAction instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.IContinueAsNewAction): temporal.omes.kitchen_sink.ContinueAsNewAction;

                /**
                 * Encodes the specified ContinueAsNewAction message. Does not implicitly {@link temporal.omes.kitchen_sink.ContinueAsNewAction.verify|verify} messages.
                 * @param message ContinueAsNewAction message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.IContinueAsNewAction, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified ContinueAsNewAction message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.ContinueAsNewAction.verify|verify} messages.
                 * @param message ContinueAsNewAction message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.IContinueAsNewAction, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a ContinueAsNewAction message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns ContinueAsNewAction
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.ContinueAsNewAction;

                /**
                 * Decodes a ContinueAsNewAction message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns ContinueAsNewAction
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.ContinueAsNewAction;

                /**
                 * Creates a ContinueAsNewAction message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns ContinueAsNewAction
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.ContinueAsNewAction;

                /**
                 * Creates a plain object from a ContinueAsNewAction message. Also converts values to other types if specified.
                 * @param message ContinueAsNewAction
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.ContinueAsNewAction, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this ContinueAsNewAction to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for ContinueAsNewAction
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            /** Properties of a RemoteActivityOptions. */
            interface IRemoteActivityOptions {

                /** Defines how the workflow will wait (or not) for cancellation of the activity to be confirmed */
                cancellationType?: (temporal.omes.kitchen_sink.ActivityCancellationType|null);

                /**
                 * If set, the worker will not tell the service that it can immediately start executing this
                 * activity. When unset/default, workers will always attempt to do so if activity execution
                 * slots are available.
                 */
                doNotEagerlyExecute?: (boolean|null);

                /** Whether this activity should run on a worker with a compatible build id or not. */
                versioningIntent?: (temporal.omes.kitchen_sink.VersioningIntent|null);
            }

            /** Represents a RemoteActivityOptions. */
            class RemoteActivityOptions implements IRemoteActivityOptions {

                /**
                 * Constructs a new RemoteActivityOptions.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.IRemoteActivityOptions);

                /** Defines how the workflow will wait (or not) for cancellation of the activity to be confirmed */
                public cancellationType: temporal.omes.kitchen_sink.ActivityCancellationType;

                /**
                 * If set, the worker will not tell the service that it can immediately start executing this
                 * activity. When unset/default, workers will always attempt to do so if activity execution
                 * slots are available.
                 */
                public doNotEagerlyExecute: boolean;

                /** Whether this activity should run on a worker with a compatible build id or not. */
                public versioningIntent: temporal.omes.kitchen_sink.VersioningIntent;

                /**
                 * Creates a new RemoteActivityOptions instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns RemoteActivityOptions instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.IRemoteActivityOptions): temporal.omes.kitchen_sink.RemoteActivityOptions;

                /**
                 * Encodes the specified RemoteActivityOptions message. Does not implicitly {@link temporal.omes.kitchen_sink.RemoteActivityOptions.verify|verify} messages.
                 * @param message RemoteActivityOptions message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.IRemoteActivityOptions, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified RemoteActivityOptions message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.RemoteActivityOptions.verify|verify} messages.
                 * @param message RemoteActivityOptions message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.IRemoteActivityOptions, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a RemoteActivityOptions message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns RemoteActivityOptions
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.RemoteActivityOptions;

                /**
                 * Decodes a RemoteActivityOptions message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns RemoteActivityOptions
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.RemoteActivityOptions;

                /**
                 * Creates a RemoteActivityOptions message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns RemoteActivityOptions
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.RemoteActivityOptions;

                /**
                 * Creates a plain object from a RemoteActivityOptions message. Also converts values to other types if specified.
                 * @param message RemoteActivityOptions
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.RemoteActivityOptions, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this RemoteActivityOptions to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for RemoteActivityOptions
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            /**
             * Used by the service to determine the fate of a child workflow
             * in case its parent is closed.
             */
            enum ParentClosePolicy {
                PARENT_CLOSE_POLICY_UNSPECIFIED = 0,
                PARENT_CLOSE_POLICY_TERMINATE = 1,
                PARENT_CLOSE_POLICY_ABANDON = 2,
                PARENT_CLOSE_POLICY_REQUEST_CANCEL = 3
            }

            /**
             * An indication of user's intent concerning what Build ID versioning approach should be used for
             * a specific command
             */
            enum VersioningIntent {
                UNSPECIFIED = 0,
                COMPATIBLE = 1,
                DEFAULT = 2
            }

            /** Controls at which point to report back to lang when a child workflow is cancelled */
            enum ChildWorkflowCancellationType {
                CHILD_WF_ABANDON = 0,
                CHILD_WF_TRY_CANCEL = 1,
                CHILD_WF_WAIT_CANCELLATION_COMPLETED = 2,
                CHILD_WF_WAIT_CANCELLATION_REQUESTED = 3
            }

            /** ActivityCancellationType enum. */
            enum ActivityCancellationType {
                TRY_CANCEL = 0,
                WAIT_CANCELLATION_COMPLETED = 1,
                ABANDON = 2
            }

            /** Properties of an ExecuteNexusOperation. */
            interface IExecuteNexusOperation {

                /** ExecuteNexusOperation endpoint */
                endpoint?: (string|null);

                /** Operation name to call */
                operation?: (string|null);

                /** Input payload for the operation */
                input?: (string|null);

                /** Headers to send with the operation */
                headers?: ({ [k: string]: string }|null);

                /** How to await on the operation */
                awaitableChoice?: (temporal.omes.kitchen_sink.IAwaitableChoice|null);

                /** Expected output for verification */
                expectedOutput?: (string|null);

                /** Actions to execute before returning from the handler workflow */
                beforeActions?: (temporal.omes.kitchen_sink.IActionSet[]|null);
            }

            /** Execute a Nexus operation */
            class ExecuteNexusOperation implements IExecuteNexusOperation {

                /**
                 * Constructs a new ExecuteNexusOperation.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.IExecuteNexusOperation);

                /** ExecuteNexusOperation endpoint. */
                public endpoint: string;

                /** Operation name to call */
                public operation: string;

                /** Input payload for the operation */
                public input: string;

                /** Headers to send with the operation */
                public headers: { [k: string]: string };

                /** How to await on the operation */
                public awaitableChoice?: (temporal.omes.kitchen_sink.IAwaitableChoice|null);

                /** Expected output for verification */
                public expectedOutput: string;

                /** Actions to execute before returning from the handler workflow */
                public beforeActions: temporal.omes.kitchen_sink.IActionSet[];

                /**
                 * Creates a new ExecuteNexusOperation instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns ExecuteNexusOperation instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.IExecuteNexusOperation): temporal.omes.kitchen_sink.ExecuteNexusOperation;

                /**
                 * Encodes the specified ExecuteNexusOperation message. Does not implicitly {@link temporal.omes.kitchen_sink.ExecuteNexusOperation.verify|verify} messages.
                 * @param message ExecuteNexusOperation message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.IExecuteNexusOperation, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified ExecuteNexusOperation message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.ExecuteNexusOperation.verify|verify} messages.
                 * @param message ExecuteNexusOperation message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.IExecuteNexusOperation, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes an ExecuteNexusOperation message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns ExecuteNexusOperation
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.ExecuteNexusOperation;

                /**
                 * Decodes an ExecuteNexusOperation message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns ExecuteNexusOperation
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.ExecuteNexusOperation;

                /**
                 * Creates an ExecuteNexusOperation message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns ExecuteNexusOperation
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.ExecuteNexusOperation;

                /**
                 * Creates a plain object from an ExecuteNexusOperation message. Also converts values to other types if specified.
                 * @param message ExecuteNexusOperation
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.ExecuteNexusOperation, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this ExecuteNexusOperation to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for ExecuteNexusOperation
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            /** Properties of a NexusHandlerInput. */
            interface INexusHandlerInput {

                /** NexusHandlerInput input */
                input?: (string|null);

                /** NexusHandlerInput beforeActions */
                beforeActions?: (temporal.omes.kitchen_sink.IActionSet[]|null);
            }

            /** Input for the Nexus handler workflow that backs echo-sync and echo-async operations */
            class NexusHandlerInput implements INexusHandlerInput {

                /**
                 * Constructs a new NexusHandlerInput.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: temporal.omes.kitchen_sink.INexusHandlerInput);

                /** NexusHandlerInput input. */
                public input: string;

                /** NexusHandlerInput beforeActions. */
                public beforeActions: temporal.omes.kitchen_sink.IActionSet[];

                /**
                 * Creates a new NexusHandlerInput instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns NexusHandlerInput instance
                 */
                public static create(properties?: temporal.omes.kitchen_sink.INexusHandlerInput): temporal.omes.kitchen_sink.NexusHandlerInput;

                /**
                 * Encodes the specified NexusHandlerInput message. Does not implicitly {@link temporal.omes.kitchen_sink.NexusHandlerInput.verify|verify} messages.
                 * @param message NexusHandlerInput message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: temporal.omes.kitchen_sink.INexusHandlerInput, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified NexusHandlerInput message, length delimited. Does not implicitly {@link temporal.omes.kitchen_sink.NexusHandlerInput.verify|verify} messages.
                 * @param message NexusHandlerInput message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: temporal.omes.kitchen_sink.INexusHandlerInput, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a NexusHandlerInput message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns NexusHandlerInput
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.omes.kitchen_sink.NexusHandlerInput;

                /**
                 * Decodes a NexusHandlerInput message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns NexusHandlerInput
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.omes.kitchen_sink.NexusHandlerInput;

                /**
                 * Creates a NexusHandlerInput message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns NexusHandlerInput
                 */
                public static fromObject(object: { [k: string]: any }): temporal.omes.kitchen_sink.NexusHandlerInput;

                /**
                 * Creates a plain object from a NexusHandlerInput message. Also converts values to other types if specified.
                 * @param message NexusHandlerInput
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: temporal.omes.kitchen_sink.NexusHandlerInput, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this NexusHandlerInput to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for NexusHandlerInput
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }
        }
    }

    /** Namespace api. */
    namespace api {

        /** Namespace common. */
        namespace common {

            /** Namespace v1. */
            namespace v1 {

                /** Properties of a DataBlob. */
                interface IDataBlob {

                    /** DataBlob encodingType */
                    encodingType?: (temporal.api.enums.v1.EncodingType|null);

                    /** DataBlob data */
                    data?: (Uint8Array|null);
                }

                /** Represents a DataBlob. */
                class DataBlob implements IDataBlob {

                    /**
                     * Constructs a new DataBlob.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.api.common.v1.IDataBlob);

                    /** DataBlob encodingType. */
                    public encodingType: temporal.api.enums.v1.EncodingType;

                    /** DataBlob data. */
                    public data: Uint8Array;

                    /**
                     * Creates a new DataBlob instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns DataBlob instance
                     */
                    public static create(properties?: temporal.api.common.v1.IDataBlob): temporal.api.common.v1.DataBlob;

                    /**
                     * Encodes the specified DataBlob message. Does not implicitly {@link temporal.api.common.v1.DataBlob.verify|verify} messages.
                     * @param message DataBlob message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.api.common.v1.IDataBlob, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified DataBlob message, length delimited. Does not implicitly {@link temporal.api.common.v1.DataBlob.verify|verify} messages.
                     * @param message DataBlob message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.api.common.v1.IDataBlob, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a DataBlob message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns DataBlob
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.common.v1.DataBlob;

                    /**
                     * Decodes a DataBlob message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns DataBlob
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.common.v1.DataBlob;

                    /**
                     * Creates a DataBlob message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns DataBlob
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.api.common.v1.DataBlob;

                    /**
                     * Creates a plain object from a DataBlob message. Also converts values to other types if specified.
                     * @param message DataBlob
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.api.common.v1.DataBlob, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this DataBlob to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for DataBlob
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of a Payloads. */
                interface IPayloads {

                    /** Payloads payloads */
                    payloads?: (temporal.api.common.v1.IPayload[]|null);
                }

                /** See `Payload` */
                class Payloads implements IPayloads {

                    /**
                     * Constructs a new Payloads.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.api.common.v1.IPayloads);

                    /** Payloads payloads. */
                    public payloads: temporal.api.common.v1.IPayload[];

                    /**
                     * Creates a new Payloads instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns Payloads instance
                     */
                    public static create(properties?: temporal.api.common.v1.IPayloads): temporal.api.common.v1.Payloads;

                    /**
                     * Encodes the specified Payloads message. Does not implicitly {@link temporal.api.common.v1.Payloads.verify|verify} messages.
                     * @param message Payloads message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.api.common.v1.IPayloads, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified Payloads message, length delimited. Does not implicitly {@link temporal.api.common.v1.Payloads.verify|verify} messages.
                     * @param message Payloads message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.api.common.v1.IPayloads, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a Payloads message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns Payloads
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.common.v1.Payloads;

                    /**
                     * Decodes a Payloads message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns Payloads
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.common.v1.Payloads;

                    /**
                     * Creates a Payloads message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns Payloads
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.api.common.v1.Payloads;

                    /**
                     * Creates a plain object from a Payloads message. Also converts values to other types if specified.
                     * @param message Payloads
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.api.common.v1.Payloads, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this Payloads to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for Payloads
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of a Payload. */
                interface IPayload {

                    /** Payload metadata */
                    metadata?: ({ [k: string]: Uint8Array }|null);

                    /** Payload data */
                    data?: (Uint8Array|null);
                }

                /**
                 * Represents some binary (byte array) data (ex: activity input parameters or workflow result) with
                 * metadata which describes this binary data (format, encoding, encryption, etc). Serialization
                 * of the data may be user-defined.
                 */
                class Payload implements IPayload {

                    /**
                     * Constructs a new Payload.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.api.common.v1.IPayload);

                    /** Payload metadata. */
                    public metadata: { [k: string]: Uint8Array };

                    /** Payload data. */
                    public data: Uint8Array;

                    /**
                     * Creates a new Payload instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns Payload instance
                     */
                    public static create(properties?: temporal.api.common.v1.IPayload): temporal.api.common.v1.Payload;

                    /**
                     * Encodes the specified Payload message. Does not implicitly {@link temporal.api.common.v1.Payload.verify|verify} messages.
                     * @param message Payload message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.api.common.v1.IPayload, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified Payload message, length delimited. Does not implicitly {@link temporal.api.common.v1.Payload.verify|verify} messages.
                     * @param message Payload message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.api.common.v1.IPayload, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a Payload message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns Payload
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.common.v1.Payload;

                    /**
                     * Decodes a Payload message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns Payload
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.common.v1.Payload;

                    /**
                     * Creates a Payload message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns Payload
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.api.common.v1.Payload;

                    /**
                     * Creates a plain object from a Payload message. Also converts values to other types if specified.
                     * @param message Payload
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.api.common.v1.Payload, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this Payload to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for Payload
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of a SearchAttributes. */
                interface ISearchAttributes {

                    /** SearchAttributes indexedFields */
                    indexedFields?: ({ [k: string]: temporal.api.common.v1.IPayload }|null);
                }

                /**
                 * A user-defined set of *indexed* fields that are used/exposed when listing/searching workflows.
                 * The payload is not serialized in a user-defined way.
                 */
                class SearchAttributes implements ISearchAttributes {

                    /**
                     * Constructs a new SearchAttributes.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.api.common.v1.ISearchAttributes);

                    /** SearchAttributes indexedFields. */
                    public indexedFields: { [k: string]: temporal.api.common.v1.IPayload };

                    /**
                     * Creates a new SearchAttributes instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns SearchAttributes instance
                     */
                    public static create(properties?: temporal.api.common.v1.ISearchAttributes): temporal.api.common.v1.SearchAttributes;

                    /**
                     * Encodes the specified SearchAttributes message. Does not implicitly {@link temporal.api.common.v1.SearchAttributes.verify|verify} messages.
                     * @param message SearchAttributes message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.api.common.v1.ISearchAttributes, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified SearchAttributes message, length delimited. Does not implicitly {@link temporal.api.common.v1.SearchAttributes.verify|verify} messages.
                     * @param message SearchAttributes message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.api.common.v1.ISearchAttributes, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a SearchAttributes message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns SearchAttributes
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.common.v1.SearchAttributes;

                    /**
                     * Decodes a SearchAttributes message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns SearchAttributes
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.common.v1.SearchAttributes;

                    /**
                     * Creates a SearchAttributes message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns SearchAttributes
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.api.common.v1.SearchAttributes;

                    /**
                     * Creates a plain object from a SearchAttributes message. Also converts values to other types if specified.
                     * @param message SearchAttributes
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.api.common.v1.SearchAttributes, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this SearchAttributes to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for SearchAttributes
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of a Memo. */
                interface IMemo {

                    /** Memo fields */
                    fields?: ({ [k: string]: temporal.api.common.v1.IPayload }|null);
                }

                /** A user-defined set of *unindexed* fields that are exposed when listing/searching workflows */
                class Memo implements IMemo {

                    /**
                     * Constructs a new Memo.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.api.common.v1.IMemo);

                    /** Memo fields. */
                    public fields: { [k: string]: temporal.api.common.v1.IPayload };

                    /**
                     * Creates a new Memo instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns Memo instance
                     */
                    public static create(properties?: temporal.api.common.v1.IMemo): temporal.api.common.v1.Memo;

                    /**
                     * Encodes the specified Memo message. Does not implicitly {@link temporal.api.common.v1.Memo.verify|verify} messages.
                     * @param message Memo message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.api.common.v1.IMemo, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified Memo message, length delimited. Does not implicitly {@link temporal.api.common.v1.Memo.verify|verify} messages.
                     * @param message Memo message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.api.common.v1.IMemo, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a Memo message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns Memo
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.common.v1.Memo;

                    /**
                     * Decodes a Memo message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns Memo
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.common.v1.Memo;

                    /**
                     * Creates a Memo message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns Memo
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.api.common.v1.Memo;

                    /**
                     * Creates a plain object from a Memo message. Also converts values to other types if specified.
                     * @param message Memo
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.api.common.v1.Memo, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this Memo to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for Memo
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of a Header. */
                interface IHeader {

                    /** Header fields */
                    fields?: ({ [k: string]: temporal.api.common.v1.IPayload }|null);
                }

                /**
                 * Contains metadata that can be attached to a variety of requests, like starting a workflow, and
                 * can be propagated between, for example, workflows and activities.
                 */
                class Header implements IHeader {

                    /**
                     * Constructs a new Header.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.api.common.v1.IHeader);

                    /** Header fields. */
                    public fields: { [k: string]: temporal.api.common.v1.IPayload };

                    /**
                     * Creates a new Header instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns Header instance
                     */
                    public static create(properties?: temporal.api.common.v1.IHeader): temporal.api.common.v1.Header;

                    /**
                     * Encodes the specified Header message. Does not implicitly {@link temporal.api.common.v1.Header.verify|verify} messages.
                     * @param message Header message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.api.common.v1.IHeader, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified Header message, length delimited. Does not implicitly {@link temporal.api.common.v1.Header.verify|verify} messages.
                     * @param message Header message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.api.common.v1.IHeader, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a Header message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns Header
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.common.v1.Header;

                    /**
                     * Decodes a Header message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns Header
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.common.v1.Header;

                    /**
                     * Creates a Header message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns Header
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.api.common.v1.Header;

                    /**
                     * Creates a plain object from a Header message. Also converts values to other types if specified.
                     * @param message Header
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.api.common.v1.Header, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this Header to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for Header
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of a WorkflowExecution. */
                interface IWorkflowExecution {

                    /** WorkflowExecution workflowId */
                    workflowId?: (string|null);

                    /** WorkflowExecution runId */
                    runId?: (string|null);
                }

                /**
                 * Identifies a specific workflow within a namespace. Practically speaking, because run_id is a
                 * uuid, a workflow execution is globally unique. Note that many commands allow specifying an empty
                 * run id as a way of saying "target the latest run of the workflow".
                 */
                class WorkflowExecution implements IWorkflowExecution {

                    /**
                     * Constructs a new WorkflowExecution.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.api.common.v1.IWorkflowExecution);

                    /** WorkflowExecution workflowId. */
                    public workflowId: string;

                    /** WorkflowExecution runId. */
                    public runId: string;

                    /**
                     * Creates a new WorkflowExecution instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns WorkflowExecution instance
                     */
                    public static create(properties?: temporal.api.common.v1.IWorkflowExecution): temporal.api.common.v1.WorkflowExecution;

                    /**
                     * Encodes the specified WorkflowExecution message. Does not implicitly {@link temporal.api.common.v1.WorkflowExecution.verify|verify} messages.
                     * @param message WorkflowExecution message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.api.common.v1.IWorkflowExecution, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified WorkflowExecution message, length delimited. Does not implicitly {@link temporal.api.common.v1.WorkflowExecution.verify|verify} messages.
                     * @param message WorkflowExecution message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.api.common.v1.IWorkflowExecution, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a WorkflowExecution message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns WorkflowExecution
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.common.v1.WorkflowExecution;

                    /**
                     * Decodes a WorkflowExecution message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns WorkflowExecution
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.common.v1.WorkflowExecution;

                    /**
                     * Creates a WorkflowExecution message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns WorkflowExecution
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.api.common.v1.WorkflowExecution;

                    /**
                     * Creates a plain object from a WorkflowExecution message. Also converts values to other types if specified.
                     * @param message WorkflowExecution
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.api.common.v1.WorkflowExecution, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this WorkflowExecution to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for WorkflowExecution
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of a WorkflowType. */
                interface IWorkflowType {

                    /** WorkflowType name */
                    name?: (string|null);
                }

                /**
                 * Represents the identifier used by a workflow author to define the workflow. Typically, the
                 * name of a function. This is sometimes referred to as the workflow's "name"
                 */
                class WorkflowType implements IWorkflowType {

                    /**
                     * Constructs a new WorkflowType.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.api.common.v1.IWorkflowType);

                    /** WorkflowType name. */
                    public name: string;

                    /**
                     * Creates a new WorkflowType instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns WorkflowType instance
                     */
                    public static create(properties?: temporal.api.common.v1.IWorkflowType): temporal.api.common.v1.WorkflowType;

                    /**
                     * Encodes the specified WorkflowType message. Does not implicitly {@link temporal.api.common.v1.WorkflowType.verify|verify} messages.
                     * @param message WorkflowType message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.api.common.v1.IWorkflowType, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified WorkflowType message, length delimited. Does not implicitly {@link temporal.api.common.v1.WorkflowType.verify|verify} messages.
                     * @param message WorkflowType message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.api.common.v1.IWorkflowType, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a WorkflowType message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns WorkflowType
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.common.v1.WorkflowType;

                    /**
                     * Decodes a WorkflowType message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns WorkflowType
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.common.v1.WorkflowType;

                    /**
                     * Creates a WorkflowType message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns WorkflowType
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.api.common.v1.WorkflowType;

                    /**
                     * Creates a plain object from a WorkflowType message. Also converts values to other types if specified.
                     * @param message WorkflowType
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.api.common.v1.WorkflowType, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this WorkflowType to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for WorkflowType
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of an ActivityType. */
                interface IActivityType {

                    /** ActivityType name */
                    name?: (string|null);
                }

                /**
                 * Represents the identifier used by a activity author to define the activity. Typically, the
                 * name of a function. This is sometimes referred to as the activity's "name"
                 */
                class ActivityType implements IActivityType {

                    /**
                     * Constructs a new ActivityType.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.api.common.v1.IActivityType);

                    /** ActivityType name. */
                    public name: string;

                    /**
                     * Creates a new ActivityType instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns ActivityType instance
                     */
                    public static create(properties?: temporal.api.common.v1.IActivityType): temporal.api.common.v1.ActivityType;

                    /**
                     * Encodes the specified ActivityType message. Does not implicitly {@link temporal.api.common.v1.ActivityType.verify|verify} messages.
                     * @param message ActivityType message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.api.common.v1.IActivityType, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified ActivityType message, length delimited. Does not implicitly {@link temporal.api.common.v1.ActivityType.verify|verify} messages.
                     * @param message ActivityType message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.api.common.v1.IActivityType, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes an ActivityType message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns ActivityType
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.common.v1.ActivityType;

                    /**
                     * Decodes an ActivityType message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns ActivityType
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.common.v1.ActivityType;

                    /**
                     * Creates an ActivityType message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns ActivityType
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.api.common.v1.ActivityType;

                    /**
                     * Creates a plain object from an ActivityType message. Also converts values to other types if specified.
                     * @param message ActivityType
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.api.common.v1.ActivityType, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this ActivityType to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for ActivityType
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of a RetryPolicy. */
                interface IRetryPolicy {

                    /** Interval of the first retry. If retryBackoffCoefficient is 1.0 then it is used for all retries. */
                    initialInterval?: (google.protobuf.IDuration|null);

                    /**
                     * Coefficient used to calculate the next retry interval.
                     * The next retry interval is previous interval multiplied by the coefficient.
                     * Must be 1 or larger.
                     */
                    backoffCoefficient?: (number|null);

                    /**
                     * Maximum interval between retries. Exponential backoff leads to interval increase.
                     * This value is the cap of the increase. Default is 100x of the initial interval.
                     */
                    maximumInterval?: (google.protobuf.IDuration|null);

                    /**
                     * Maximum number of attempts. When exceeded the retries stop even if not expired yet.
                     * 1 disables retries. 0 means unlimited (up to the timeouts)
                     */
                    maximumAttempts?: (number|null);

                    /**
                     * Non-Retryable errors types. Will stop retrying if the error type matches this list. Note that
                     * this is not a substring match, the error *type* (not message) must match exactly.
                     */
                    nonRetryableErrorTypes?: (string[]|null);
                }

                /** How retries ought to be handled, usable by both workflows and activities */
                class RetryPolicy implements IRetryPolicy {

                    /**
                     * Constructs a new RetryPolicy.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.api.common.v1.IRetryPolicy);

                    /** Interval of the first retry. If retryBackoffCoefficient is 1.0 then it is used for all retries. */
                    public initialInterval?: (google.protobuf.IDuration|null);

                    /**
                     * Coefficient used to calculate the next retry interval.
                     * The next retry interval is previous interval multiplied by the coefficient.
                     * Must be 1 or larger.
                     */
                    public backoffCoefficient: number;

                    /**
                     * Maximum interval between retries. Exponential backoff leads to interval increase.
                     * This value is the cap of the increase. Default is 100x of the initial interval.
                     */
                    public maximumInterval?: (google.protobuf.IDuration|null);

                    /**
                     * Maximum number of attempts. When exceeded the retries stop even if not expired yet.
                     * 1 disables retries. 0 means unlimited (up to the timeouts)
                     */
                    public maximumAttempts: number;

                    /**
                     * Non-Retryable errors types. Will stop retrying if the error type matches this list. Note that
                     * this is not a substring match, the error *type* (not message) must match exactly.
                     */
                    public nonRetryableErrorTypes: string[];

                    /**
                     * Creates a new RetryPolicy instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns RetryPolicy instance
                     */
                    public static create(properties?: temporal.api.common.v1.IRetryPolicy): temporal.api.common.v1.RetryPolicy;

                    /**
                     * Encodes the specified RetryPolicy message. Does not implicitly {@link temporal.api.common.v1.RetryPolicy.verify|verify} messages.
                     * @param message RetryPolicy message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.api.common.v1.IRetryPolicy, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified RetryPolicy message, length delimited. Does not implicitly {@link temporal.api.common.v1.RetryPolicy.verify|verify} messages.
                     * @param message RetryPolicy message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.api.common.v1.IRetryPolicy, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a RetryPolicy message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns RetryPolicy
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.common.v1.RetryPolicy;

                    /**
                     * Decodes a RetryPolicy message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns RetryPolicy
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.common.v1.RetryPolicy;

                    /**
                     * Creates a RetryPolicy message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns RetryPolicy
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.api.common.v1.RetryPolicy;

                    /**
                     * Creates a plain object from a RetryPolicy message. Also converts values to other types if specified.
                     * @param message RetryPolicy
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.api.common.v1.RetryPolicy, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this RetryPolicy to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for RetryPolicy
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of a MeteringMetadata. */
                interface IMeteringMetadata {

                    /**
                     * Count of local activities which have begun an execution attempt during this workflow task,
                     * and whose first attempt occurred in some previous task. This is used for metering
                     * purposes, and does not affect workflow state.
                     *
                     * (-- api-linter: core::0141::forbidden-types=disabled
                     * aip.dev/not-precedent: Negative values make no sense to represent. --)
                     */
                    nonfirstLocalActivityExecutionAttempts?: (number|null);
                }

                /** Metadata relevant for metering purposes */
                class MeteringMetadata implements IMeteringMetadata {

                    /**
                     * Constructs a new MeteringMetadata.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.api.common.v1.IMeteringMetadata);

                    /**
                     * Count of local activities which have begun an execution attempt during this workflow task,
                     * and whose first attempt occurred in some previous task. This is used for metering
                     * purposes, and does not affect workflow state.
                     *
                     * (-- api-linter: core::0141::forbidden-types=disabled
                     * aip.dev/not-precedent: Negative values make no sense to represent. --)
                     */
                    public nonfirstLocalActivityExecutionAttempts: number;

                    /**
                     * Creates a new MeteringMetadata instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns MeteringMetadata instance
                     */
                    public static create(properties?: temporal.api.common.v1.IMeteringMetadata): temporal.api.common.v1.MeteringMetadata;

                    /**
                     * Encodes the specified MeteringMetadata message. Does not implicitly {@link temporal.api.common.v1.MeteringMetadata.verify|verify} messages.
                     * @param message MeteringMetadata message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.api.common.v1.IMeteringMetadata, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified MeteringMetadata message, length delimited. Does not implicitly {@link temporal.api.common.v1.MeteringMetadata.verify|verify} messages.
                     * @param message MeteringMetadata message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.api.common.v1.IMeteringMetadata, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a MeteringMetadata message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns MeteringMetadata
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.common.v1.MeteringMetadata;

                    /**
                     * Decodes a MeteringMetadata message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns MeteringMetadata
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.common.v1.MeteringMetadata;

                    /**
                     * Creates a MeteringMetadata message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns MeteringMetadata
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.api.common.v1.MeteringMetadata;

                    /**
                     * Creates a plain object from a MeteringMetadata message. Also converts values to other types if specified.
                     * @param message MeteringMetadata
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.api.common.v1.MeteringMetadata, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this MeteringMetadata to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for MeteringMetadata
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of a WorkerVersionStamp. */
                interface IWorkerVersionStamp {

                    /**
                     * An opaque whole-worker identifier. Replaces the deprecated `binary_checksum` field when this
                     * message is included in requests which previously used that.
                     */
                    buildId?: (string|null);

                    /**
                     * If set, the worker is opting in to worker versioning. Otherwise, this is used only as a
                     * marker for workflow reset points and the BuildIDs search attribute.
                     */
                    useVersioning?: (boolean|null);
                }

                /**
                 * Deprecated. This message is replaced with `Deployment` and `VersioningBehavior`.
                 * Identifies the version(s) of a worker that processed a task
                 */
                class WorkerVersionStamp implements IWorkerVersionStamp {

                    /**
                     * Constructs a new WorkerVersionStamp.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.api.common.v1.IWorkerVersionStamp);

                    /**
                     * An opaque whole-worker identifier. Replaces the deprecated `binary_checksum` field when this
                     * message is included in requests which previously used that.
                     */
                    public buildId: string;

                    /**
                     * If set, the worker is opting in to worker versioning. Otherwise, this is used only as a
                     * marker for workflow reset points and the BuildIDs search attribute.
                     */
                    public useVersioning: boolean;

                    /**
                     * Creates a new WorkerVersionStamp instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns WorkerVersionStamp instance
                     */
                    public static create(properties?: temporal.api.common.v1.IWorkerVersionStamp): temporal.api.common.v1.WorkerVersionStamp;

                    /**
                     * Encodes the specified WorkerVersionStamp message. Does not implicitly {@link temporal.api.common.v1.WorkerVersionStamp.verify|verify} messages.
                     * @param message WorkerVersionStamp message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.api.common.v1.IWorkerVersionStamp, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified WorkerVersionStamp message, length delimited. Does not implicitly {@link temporal.api.common.v1.WorkerVersionStamp.verify|verify} messages.
                     * @param message WorkerVersionStamp message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.api.common.v1.IWorkerVersionStamp, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a WorkerVersionStamp message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns WorkerVersionStamp
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.common.v1.WorkerVersionStamp;

                    /**
                     * Decodes a WorkerVersionStamp message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns WorkerVersionStamp
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.common.v1.WorkerVersionStamp;

                    /**
                     * Creates a WorkerVersionStamp message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns WorkerVersionStamp
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.api.common.v1.WorkerVersionStamp;

                    /**
                     * Creates a plain object from a WorkerVersionStamp message. Also converts values to other types if specified.
                     * @param message WorkerVersionStamp
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.api.common.v1.WorkerVersionStamp, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this WorkerVersionStamp to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for WorkerVersionStamp
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of a WorkerVersionCapabilities. */
                interface IWorkerVersionCapabilities {

                    /** An opaque whole-worker identifier */
                    buildId?: (string|null);

                    /**
                     * If set, the worker is opting in to worker versioning, and wishes to only receive appropriate
                     * tasks.
                     */
                    useVersioning?: (boolean|null);

                    /** Must be sent if user has set a deployment series name (versioning-3). */
                    deploymentSeriesName?: (string|null);
                }

                /**
                 * Identifies the version that a worker is compatible with when polling or identifying itself,
                 * and whether or not this worker is opting into the build-id based versioning feature. This is
                 * used by matching to determine which workers ought to receive what tasks.
                 * Deprecated. Use WorkerDeploymentOptions instead.
                 */
                class WorkerVersionCapabilities implements IWorkerVersionCapabilities {

                    /**
                     * Constructs a new WorkerVersionCapabilities.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.api.common.v1.IWorkerVersionCapabilities);

                    /** An opaque whole-worker identifier */
                    public buildId: string;

                    /**
                     * If set, the worker is opting in to worker versioning, and wishes to only receive appropriate
                     * tasks.
                     */
                    public useVersioning: boolean;

                    /** Must be sent if user has set a deployment series name (versioning-3). */
                    public deploymentSeriesName: string;

                    /**
                     * Creates a new WorkerVersionCapabilities instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns WorkerVersionCapabilities instance
                     */
                    public static create(properties?: temporal.api.common.v1.IWorkerVersionCapabilities): temporal.api.common.v1.WorkerVersionCapabilities;

                    /**
                     * Encodes the specified WorkerVersionCapabilities message. Does not implicitly {@link temporal.api.common.v1.WorkerVersionCapabilities.verify|verify} messages.
                     * @param message WorkerVersionCapabilities message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.api.common.v1.IWorkerVersionCapabilities, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified WorkerVersionCapabilities message, length delimited. Does not implicitly {@link temporal.api.common.v1.WorkerVersionCapabilities.verify|verify} messages.
                     * @param message WorkerVersionCapabilities message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.api.common.v1.IWorkerVersionCapabilities, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a WorkerVersionCapabilities message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns WorkerVersionCapabilities
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.common.v1.WorkerVersionCapabilities;

                    /**
                     * Decodes a WorkerVersionCapabilities message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns WorkerVersionCapabilities
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.common.v1.WorkerVersionCapabilities;

                    /**
                     * Creates a WorkerVersionCapabilities message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns WorkerVersionCapabilities
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.api.common.v1.WorkerVersionCapabilities;

                    /**
                     * Creates a plain object from a WorkerVersionCapabilities message. Also converts values to other types if specified.
                     * @param message WorkerVersionCapabilities
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.api.common.v1.WorkerVersionCapabilities, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this WorkerVersionCapabilities to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for WorkerVersionCapabilities
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of a ResetOptions. */
                interface IResetOptions {

                    /** Resets to the first workflow task completed or started event. */
                    firstWorkflowTask?: (google.protobuf.IEmpty|null);

                    /** Resets to the last workflow task completed or started event. */
                    lastWorkflowTask?: (google.protobuf.IEmpty|null);

                    /**
                     * The id of a specific `WORKFLOW_TASK_COMPLETED`,`WORKFLOW_TASK_TIMED_OUT`, `WORKFLOW_TASK_FAILED`, or
                     * `WORKFLOW_TASK_STARTED` event to reset to.
                     * Note that this option doesn't make sense when used as part of a batch request.
                     */
                    workflowTaskId?: (Long|null);

                    /**
                     * Resets to the first workflow task processed by this build id.
                     * If the workflow was not processed by the build id, or the workflow task can't be
                     * determined, no reset will be performed.
                     * Note that by default, this reset is allowed to be to a prior run in a chain of
                     * continue-as-new.
                     */
                    buildId?: (string|null);

                    /**
                     * Deprecated. Use `options`.
                     * Default: RESET_REAPPLY_TYPE_SIGNAL
                     */
                    resetReapplyType?: (temporal.api.enums.v1.ResetReapplyType|null);

                    /**
                     * If true, limit the reset to only within the current run. (Applies to build_id targets and
                     * possibly others in the future.)
                     */
                    currentRunOnly?: (boolean|null);

                    /** Event types not to be reapplied */
                    resetReapplyExcludeTypes?: (temporal.api.enums.v1.ResetReapplyExcludeType[]|null);
                }

                /**
                 * Describes where and how to reset a workflow, used for batch reset currently
                 * and may be used for single-workflow reset later.
                 */
                class ResetOptions implements IResetOptions {

                    /**
                     * Constructs a new ResetOptions.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.api.common.v1.IResetOptions);

                    /** Resets to the first workflow task completed or started event. */
                    public firstWorkflowTask?: (google.protobuf.IEmpty|null);

                    /** Resets to the last workflow task completed or started event. */
                    public lastWorkflowTask?: (google.protobuf.IEmpty|null);

                    /**
                     * The id of a specific `WORKFLOW_TASK_COMPLETED`,`WORKFLOW_TASK_TIMED_OUT`, `WORKFLOW_TASK_FAILED`, or
                     * `WORKFLOW_TASK_STARTED` event to reset to.
                     * Note that this option doesn't make sense when used as part of a batch request.
                     */
                    public workflowTaskId?: (Long|null);

                    /**
                     * Resets to the first workflow task processed by this build id.
                     * If the workflow was not processed by the build id, or the workflow task can't be
                     * determined, no reset will be performed.
                     * Note that by default, this reset is allowed to be to a prior run in a chain of
                     * continue-as-new.
                     */
                    public buildId?: (string|null);

                    /**
                     * Deprecated. Use `options`.
                     * Default: RESET_REAPPLY_TYPE_SIGNAL
                     */
                    public resetReapplyType: temporal.api.enums.v1.ResetReapplyType;

                    /**
                     * If true, limit the reset to only within the current run. (Applies to build_id targets and
                     * possibly others in the future.)
                     */
                    public currentRunOnly: boolean;

                    /** Event types not to be reapplied */
                    public resetReapplyExcludeTypes: temporal.api.enums.v1.ResetReapplyExcludeType[];

                    /** Which workflow task to reset to. */
                    public target?: ("firstWorkflowTask"|"lastWorkflowTask"|"workflowTaskId"|"buildId");

                    /**
                     * Creates a new ResetOptions instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns ResetOptions instance
                     */
                    public static create(properties?: temporal.api.common.v1.IResetOptions): temporal.api.common.v1.ResetOptions;

                    /**
                     * Encodes the specified ResetOptions message. Does not implicitly {@link temporal.api.common.v1.ResetOptions.verify|verify} messages.
                     * @param message ResetOptions message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.api.common.v1.IResetOptions, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified ResetOptions message, length delimited. Does not implicitly {@link temporal.api.common.v1.ResetOptions.verify|verify} messages.
                     * @param message ResetOptions message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.api.common.v1.IResetOptions, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a ResetOptions message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns ResetOptions
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.common.v1.ResetOptions;

                    /**
                     * Decodes a ResetOptions message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns ResetOptions
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.common.v1.ResetOptions;

                    /**
                     * Creates a ResetOptions message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns ResetOptions
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.api.common.v1.ResetOptions;

                    /**
                     * Creates a plain object from a ResetOptions message. Also converts values to other types if specified.
                     * @param message ResetOptions
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.api.common.v1.ResetOptions, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this ResetOptions to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for ResetOptions
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of a Callback. */
                interface ICallback {

                    /** Callback nexus */
                    nexus?: (temporal.api.common.v1.Callback.INexus|null);

                    /** Callback internal */
                    internal?: (temporal.api.common.v1.Callback.IInternal|null);

                    /**
                     * Links associated with the callback. It can be used to link to underlying resources of the
                     * callback.
                     */
                    links?: (temporal.api.common.v1.ILink[]|null);
                }

                /** Callback to attach to various events in the system, e.g. workflow run completion. */
                class Callback implements ICallback {

                    /**
                     * Constructs a new Callback.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.api.common.v1.ICallback);

                    /** Callback nexus. */
                    public nexus?: (temporal.api.common.v1.Callback.INexus|null);

                    /** Callback internal. */
                    public internal?: (temporal.api.common.v1.Callback.IInternal|null);

                    /**
                     * Links associated with the callback. It can be used to link to underlying resources of the
                     * callback.
                     */
                    public links: temporal.api.common.v1.ILink[];

                    /** Callback variant. */
                    public variant?: ("nexus"|"internal");

                    /**
                     * Creates a new Callback instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns Callback instance
                     */
                    public static create(properties?: temporal.api.common.v1.ICallback): temporal.api.common.v1.Callback;

                    /**
                     * Encodes the specified Callback message. Does not implicitly {@link temporal.api.common.v1.Callback.verify|verify} messages.
                     * @param message Callback message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.api.common.v1.ICallback, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified Callback message, length delimited. Does not implicitly {@link temporal.api.common.v1.Callback.verify|verify} messages.
                     * @param message Callback message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.api.common.v1.ICallback, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a Callback message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns Callback
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.common.v1.Callback;

                    /**
                     * Decodes a Callback message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns Callback
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.common.v1.Callback;

                    /**
                     * Creates a Callback message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns Callback
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.api.common.v1.Callback;

                    /**
                     * Creates a plain object from a Callback message. Also converts values to other types if specified.
                     * @param message Callback
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.api.common.v1.Callback, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this Callback to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for Callback
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                namespace Callback {

                    /** Properties of a Nexus. */
                    interface INexus {

                        /** Callback URL. */
                        url?: (string|null);

                        /** Header to attach to callback request. */
                        header?: ({ [k: string]: string }|null);
                    }

                    /** Represents a Nexus. */
                    class Nexus implements INexus {

                        /**
                         * Constructs a new Nexus.
                         * @param [properties] Properties to set
                         */
                        constructor(properties?: temporal.api.common.v1.Callback.INexus);

                        /** Callback URL. */
                        public url: string;

                        /** Header to attach to callback request. */
                        public header: { [k: string]: string };

                        /**
                         * Creates a new Nexus instance using the specified properties.
                         * @param [properties] Properties to set
                         * @returns Nexus instance
                         */
                        public static create(properties?: temporal.api.common.v1.Callback.INexus): temporal.api.common.v1.Callback.Nexus;

                        /**
                         * Encodes the specified Nexus message. Does not implicitly {@link temporal.api.common.v1.Callback.Nexus.verify|verify} messages.
                         * @param message Nexus message or plain object to encode
                         * @param [writer] Writer to encode to
                         * @returns Writer
                         */
                        public static encode(message: temporal.api.common.v1.Callback.INexus, writer?: $protobuf.Writer): $protobuf.Writer;

                        /**
                         * Encodes the specified Nexus message, length delimited. Does not implicitly {@link temporal.api.common.v1.Callback.Nexus.verify|verify} messages.
                         * @param message Nexus message or plain object to encode
                         * @param [writer] Writer to encode to
                         * @returns Writer
                         */
                        public static encodeDelimited(message: temporal.api.common.v1.Callback.INexus, writer?: $protobuf.Writer): $protobuf.Writer;

                        /**
                         * Decodes a Nexus message from the specified reader or buffer.
                         * @param reader Reader or buffer to decode from
                         * @param [length] Message length if known beforehand
                         * @returns Nexus
                         * @throws {Error} If the payload is not a reader or valid buffer
                         * @throws {$protobuf.util.ProtocolError} If required fields are missing
                         */
                        public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.common.v1.Callback.Nexus;

                        /**
                         * Decodes a Nexus message from the specified reader or buffer, length delimited.
                         * @param reader Reader or buffer to decode from
                         * @returns Nexus
                         * @throws {Error} If the payload is not a reader or valid buffer
                         * @throws {$protobuf.util.ProtocolError} If required fields are missing
                         */
                        public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.common.v1.Callback.Nexus;

                        /**
                         * Creates a Nexus message from a plain object. Also converts values to their respective internal types.
                         * @param object Plain object
                         * @returns Nexus
                         */
                        public static fromObject(object: { [k: string]: any }): temporal.api.common.v1.Callback.Nexus;

                        /**
                         * Creates a plain object from a Nexus message. Also converts values to other types if specified.
                         * @param message Nexus
                         * @param [options] Conversion options
                         * @returns Plain object
                         */
                        public static toObject(message: temporal.api.common.v1.Callback.Nexus, options?: $protobuf.IConversionOptions): { [k: string]: any };

                        /**
                         * Converts this Nexus to JSON.
                         * @returns JSON object
                         */
                        public toJSON(): { [k: string]: any };

                        /**
                         * Gets the default type url for Nexus
                         * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                         * @returns The default type url
                         */
                        public static getTypeUrl(typeUrlPrefix?: string): string;
                    }

                    /** Properties of an Internal. */
                    interface IInternal {

                        /** Opaque internal data. */
                        data?: (Uint8Array|null);
                    }

                    /**
                     * Callbacks to be delivered internally within the system.
                     * This variant is not settable in the API and will be rejected by the service with an INVALID_ARGUMENT error.
                     * The only reason that this is exposed is because callbacks are replicated across clusters via the
                     * WorkflowExecutionStarted event, which is defined in the public API.
                     */
                    class Internal implements IInternal {

                        /**
                         * Constructs a new Internal.
                         * @param [properties] Properties to set
                         */
                        constructor(properties?: temporal.api.common.v1.Callback.IInternal);

                        /** Opaque internal data. */
                        public data: Uint8Array;

                        /**
                         * Creates a new Internal instance using the specified properties.
                         * @param [properties] Properties to set
                         * @returns Internal instance
                         */
                        public static create(properties?: temporal.api.common.v1.Callback.IInternal): temporal.api.common.v1.Callback.Internal;

                        /**
                         * Encodes the specified Internal message. Does not implicitly {@link temporal.api.common.v1.Callback.Internal.verify|verify} messages.
                         * @param message Internal message or plain object to encode
                         * @param [writer] Writer to encode to
                         * @returns Writer
                         */
                        public static encode(message: temporal.api.common.v1.Callback.IInternal, writer?: $protobuf.Writer): $protobuf.Writer;

                        /**
                         * Encodes the specified Internal message, length delimited. Does not implicitly {@link temporal.api.common.v1.Callback.Internal.verify|verify} messages.
                         * @param message Internal message or plain object to encode
                         * @param [writer] Writer to encode to
                         * @returns Writer
                         */
                        public static encodeDelimited(message: temporal.api.common.v1.Callback.IInternal, writer?: $protobuf.Writer): $protobuf.Writer;

                        /**
                         * Decodes an Internal message from the specified reader or buffer.
                         * @param reader Reader or buffer to decode from
                         * @param [length] Message length if known beforehand
                         * @returns Internal
                         * @throws {Error} If the payload is not a reader or valid buffer
                         * @throws {$protobuf.util.ProtocolError} If required fields are missing
                         */
                        public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.common.v1.Callback.Internal;

                        /**
                         * Decodes an Internal message from the specified reader or buffer, length delimited.
                         * @param reader Reader or buffer to decode from
                         * @returns Internal
                         * @throws {Error} If the payload is not a reader or valid buffer
                         * @throws {$protobuf.util.ProtocolError} If required fields are missing
                         */
                        public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.common.v1.Callback.Internal;

                        /**
                         * Creates an Internal message from a plain object. Also converts values to their respective internal types.
                         * @param object Plain object
                         * @returns Internal
                         */
                        public static fromObject(object: { [k: string]: any }): temporal.api.common.v1.Callback.Internal;

                        /**
                         * Creates a plain object from an Internal message. Also converts values to other types if specified.
                         * @param message Internal
                         * @param [options] Conversion options
                         * @returns Plain object
                         */
                        public static toObject(message: temporal.api.common.v1.Callback.Internal, options?: $protobuf.IConversionOptions): { [k: string]: any };

                        /**
                         * Converts this Internal to JSON.
                         * @returns JSON object
                         */
                        public toJSON(): { [k: string]: any };

                        /**
                         * Gets the default type url for Internal
                         * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                         * @returns The default type url
                         */
                        public static getTypeUrl(typeUrlPrefix?: string): string;
                    }
                }

                /** Properties of a Link. */
                interface ILink {

                    /** Link workflowEvent */
                    workflowEvent?: (temporal.api.common.v1.Link.IWorkflowEvent|null);

                    /** Link batchJob */
                    batchJob?: (temporal.api.common.v1.Link.IBatchJob|null);
                }

                /**
                 * Link can be associated with history events. It might contain information about an external entity
                 * related to the history event. For example, workflow A makes a Nexus call that starts workflow B:
                 * in this case, a history event in workflow A could contain a Link to the workflow started event in
                 * workflow B, and vice-versa.
                 */
                class Link implements ILink {

                    /**
                     * Constructs a new Link.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.api.common.v1.ILink);

                    /** Link workflowEvent. */
                    public workflowEvent?: (temporal.api.common.v1.Link.IWorkflowEvent|null);

                    /** Link batchJob. */
                    public batchJob?: (temporal.api.common.v1.Link.IBatchJob|null);

                    /** Link variant. */
                    public variant?: ("workflowEvent"|"batchJob");

                    /**
                     * Creates a new Link instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns Link instance
                     */
                    public static create(properties?: temporal.api.common.v1.ILink): temporal.api.common.v1.Link;

                    /**
                     * Encodes the specified Link message. Does not implicitly {@link temporal.api.common.v1.Link.verify|verify} messages.
                     * @param message Link message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.api.common.v1.ILink, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified Link message, length delimited. Does not implicitly {@link temporal.api.common.v1.Link.verify|verify} messages.
                     * @param message Link message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.api.common.v1.ILink, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a Link message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns Link
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.common.v1.Link;

                    /**
                     * Decodes a Link message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns Link
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.common.v1.Link;

                    /**
                     * Creates a Link message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns Link
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.api.common.v1.Link;

                    /**
                     * Creates a plain object from a Link message. Also converts values to other types if specified.
                     * @param message Link
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.api.common.v1.Link, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this Link to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for Link
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                namespace Link {

                    /** Properties of a WorkflowEvent. */
                    interface IWorkflowEvent {

                        /** WorkflowEvent namespace */
                        namespace?: (string|null);

                        /** WorkflowEvent workflowId */
                        workflowId?: (string|null);

                        /** WorkflowEvent runId */
                        runId?: (string|null);

                        /** WorkflowEvent eventRef */
                        eventRef?: (temporal.api.common.v1.Link.WorkflowEvent.IEventReference|null);

                        /** WorkflowEvent requestIdRef */
                        requestIdRef?: (temporal.api.common.v1.Link.WorkflowEvent.IRequestIdReference|null);
                    }

                    /** Represents a WorkflowEvent. */
                    class WorkflowEvent implements IWorkflowEvent {

                        /**
                         * Constructs a new WorkflowEvent.
                         * @param [properties] Properties to set
                         */
                        constructor(properties?: temporal.api.common.v1.Link.IWorkflowEvent);

                        /** WorkflowEvent namespace. */
                        public namespace: string;

                        /** WorkflowEvent workflowId. */
                        public workflowId: string;

                        /** WorkflowEvent runId. */
                        public runId: string;

                        /** WorkflowEvent eventRef. */
                        public eventRef?: (temporal.api.common.v1.Link.WorkflowEvent.IEventReference|null);

                        /** WorkflowEvent requestIdRef. */
                        public requestIdRef?: (temporal.api.common.v1.Link.WorkflowEvent.IRequestIdReference|null);

                        /**
                         * Additional information about the workflow event.
                         * Eg: the caller workflow can send the history event details that made the Nexus call.
                         */
                        public reference?: ("eventRef"|"requestIdRef");

                        /**
                         * Creates a new WorkflowEvent instance using the specified properties.
                         * @param [properties] Properties to set
                         * @returns WorkflowEvent instance
                         */
                        public static create(properties?: temporal.api.common.v1.Link.IWorkflowEvent): temporal.api.common.v1.Link.WorkflowEvent;

                        /**
                         * Encodes the specified WorkflowEvent message. Does not implicitly {@link temporal.api.common.v1.Link.WorkflowEvent.verify|verify} messages.
                         * @param message WorkflowEvent message or plain object to encode
                         * @param [writer] Writer to encode to
                         * @returns Writer
                         */
                        public static encode(message: temporal.api.common.v1.Link.IWorkflowEvent, writer?: $protobuf.Writer): $protobuf.Writer;

                        /**
                         * Encodes the specified WorkflowEvent message, length delimited. Does not implicitly {@link temporal.api.common.v1.Link.WorkflowEvent.verify|verify} messages.
                         * @param message WorkflowEvent message or plain object to encode
                         * @param [writer] Writer to encode to
                         * @returns Writer
                         */
                        public static encodeDelimited(message: temporal.api.common.v1.Link.IWorkflowEvent, writer?: $protobuf.Writer): $protobuf.Writer;

                        /**
                         * Decodes a WorkflowEvent message from the specified reader or buffer.
                         * @param reader Reader or buffer to decode from
                         * @param [length] Message length if known beforehand
                         * @returns WorkflowEvent
                         * @throws {Error} If the payload is not a reader or valid buffer
                         * @throws {$protobuf.util.ProtocolError} If required fields are missing
                         */
                        public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.common.v1.Link.WorkflowEvent;

                        /**
                         * Decodes a WorkflowEvent message from the specified reader or buffer, length delimited.
                         * @param reader Reader or buffer to decode from
                         * @returns WorkflowEvent
                         * @throws {Error} If the payload is not a reader or valid buffer
                         * @throws {$protobuf.util.ProtocolError} If required fields are missing
                         */
                        public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.common.v1.Link.WorkflowEvent;

                        /**
                         * Creates a WorkflowEvent message from a plain object. Also converts values to their respective internal types.
                         * @param object Plain object
                         * @returns WorkflowEvent
                         */
                        public static fromObject(object: { [k: string]: any }): temporal.api.common.v1.Link.WorkflowEvent;

                        /**
                         * Creates a plain object from a WorkflowEvent message. Also converts values to other types if specified.
                         * @param message WorkflowEvent
                         * @param [options] Conversion options
                         * @returns Plain object
                         */
                        public static toObject(message: temporal.api.common.v1.Link.WorkflowEvent, options?: $protobuf.IConversionOptions): { [k: string]: any };

                        /**
                         * Converts this WorkflowEvent to JSON.
                         * @returns JSON object
                         */
                        public toJSON(): { [k: string]: any };

                        /**
                         * Gets the default type url for WorkflowEvent
                         * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                         * @returns The default type url
                         */
                        public static getTypeUrl(typeUrlPrefix?: string): string;
                    }

                    namespace WorkflowEvent {

                        /** Properties of an EventReference. */
                        interface IEventReference {

                            /** EventReference eventId */
                            eventId?: (Long|null);

                            /** EventReference eventType */
                            eventType?: (temporal.api.enums.v1.EventType|null);
                        }

                        /** EventReference is a direct reference to a history event through the event ID. */
                        class EventReference implements IEventReference {

                            /**
                             * Constructs a new EventReference.
                             * @param [properties] Properties to set
                             */
                            constructor(properties?: temporal.api.common.v1.Link.WorkflowEvent.IEventReference);

                            /** EventReference eventId. */
                            public eventId: Long;

                            /** EventReference eventType. */
                            public eventType: temporal.api.enums.v1.EventType;

                            /**
                             * Creates a new EventReference instance using the specified properties.
                             * @param [properties] Properties to set
                             * @returns EventReference instance
                             */
                            public static create(properties?: temporal.api.common.v1.Link.WorkflowEvent.IEventReference): temporal.api.common.v1.Link.WorkflowEvent.EventReference;

                            /**
                             * Encodes the specified EventReference message. Does not implicitly {@link temporal.api.common.v1.Link.WorkflowEvent.EventReference.verify|verify} messages.
                             * @param message EventReference message or plain object to encode
                             * @param [writer] Writer to encode to
                             * @returns Writer
                             */
                            public static encode(message: temporal.api.common.v1.Link.WorkflowEvent.IEventReference, writer?: $protobuf.Writer): $protobuf.Writer;

                            /**
                             * Encodes the specified EventReference message, length delimited. Does not implicitly {@link temporal.api.common.v1.Link.WorkflowEvent.EventReference.verify|verify} messages.
                             * @param message EventReference message or plain object to encode
                             * @param [writer] Writer to encode to
                             * @returns Writer
                             */
                            public static encodeDelimited(message: temporal.api.common.v1.Link.WorkflowEvent.IEventReference, writer?: $protobuf.Writer): $protobuf.Writer;

                            /**
                             * Decodes an EventReference message from the specified reader or buffer.
                             * @param reader Reader or buffer to decode from
                             * @param [length] Message length if known beforehand
                             * @returns EventReference
                             * @throws {Error} If the payload is not a reader or valid buffer
                             * @throws {$protobuf.util.ProtocolError} If required fields are missing
                             */
                            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.common.v1.Link.WorkflowEvent.EventReference;

                            /**
                             * Decodes an EventReference message from the specified reader or buffer, length delimited.
                             * @param reader Reader or buffer to decode from
                             * @returns EventReference
                             * @throws {Error} If the payload is not a reader or valid buffer
                             * @throws {$protobuf.util.ProtocolError} If required fields are missing
                             */
                            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.common.v1.Link.WorkflowEvent.EventReference;

                            /**
                             * Creates an EventReference message from a plain object. Also converts values to their respective internal types.
                             * @param object Plain object
                             * @returns EventReference
                             */
                            public static fromObject(object: { [k: string]: any }): temporal.api.common.v1.Link.WorkflowEvent.EventReference;

                            /**
                             * Creates a plain object from an EventReference message. Also converts values to other types if specified.
                             * @param message EventReference
                             * @param [options] Conversion options
                             * @returns Plain object
                             */
                            public static toObject(message: temporal.api.common.v1.Link.WorkflowEvent.EventReference, options?: $protobuf.IConversionOptions): { [k: string]: any };

                            /**
                             * Converts this EventReference to JSON.
                             * @returns JSON object
                             */
                            public toJSON(): { [k: string]: any };

                            /**
                             * Gets the default type url for EventReference
                             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                             * @returns The default type url
                             */
                            public static getTypeUrl(typeUrlPrefix?: string): string;
                        }

                        /** Properties of a RequestIdReference. */
                        interface IRequestIdReference {

                            /** RequestIdReference requestId */
                            requestId?: (string|null);

                            /** RequestIdReference eventType */
                            eventType?: (temporal.api.enums.v1.EventType|null);
                        }

                        /** RequestIdReference is a indirect reference to a history event through the request ID. */
                        class RequestIdReference implements IRequestIdReference {

                            /**
                             * Constructs a new RequestIdReference.
                             * @param [properties] Properties to set
                             */
                            constructor(properties?: temporal.api.common.v1.Link.WorkflowEvent.IRequestIdReference);

                            /** RequestIdReference requestId. */
                            public requestId: string;

                            /** RequestIdReference eventType. */
                            public eventType: temporal.api.enums.v1.EventType;

                            /**
                             * Creates a new RequestIdReference instance using the specified properties.
                             * @param [properties] Properties to set
                             * @returns RequestIdReference instance
                             */
                            public static create(properties?: temporal.api.common.v1.Link.WorkflowEvent.IRequestIdReference): temporal.api.common.v1.Link.WorkflowEvent.RequestIdReference;

                            /**
                             * Encodes the specified RequestIdReference message. Does not implicitly {@link temporal.api.common.v1.Link.WorkflowEvent.RequestIdReference.verify|verify} messages.
                             * @param message RequestIdReference message or plain object to encode
                             * @param [writer] Writer to encode to
                             * @returns Writer
                             */
                            public static encode(message: temporal.api.common.v1.Link.WorkflowEvent.IRequestIdReference, writer?: $protobuf.Writer): $protobuf.Writer;

                            /**
                             * Encodes the specified RequestIdReference message, length delimited. Does not implicitly {@link temporal.api.common.v1.Link.WorkflowEvent.RequestIdReference.verify|verify} messages.
                             * @param message RequestIdReference message or plain object to encode
                             * @param [writer] Writer to encode to
                             * @returns Writer
                             */
                            public static encodeDelimited(message: temporal.api.common.v1.Link.WorkflowEvent.IRequestIdReference, writer?: $protobuf.Writer): $protobuf.Writer;

                            /**
                             * Decodes a RequestIdReference message from the specified reader or buffer.
                             * @param reader Reader or buffer to decode from
                             * @param [length] Message length if known beforehand
                             * @returns RequestIdReference
                             * @throws {Error} If the payload is not a reader or valid buffer
                             * @throws {$protobuf.util.ProtocolError} If required fields are missing
                             */
                            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.common.v1.Link.WorkflowEvent.RequestIdReference;

                            /**
                             * Decodes a RequestIdReference message from the specified reader or buffer, length delimited.
                             * @param reader Reader or buffer to decode from
                             * @returns RequestIdReference
                             * @throws {Error} If the payload is not a reader or valid buffer
                             * @throws {$protobuf.util.ProtocolError} If required fields are missing
                             */
                            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.common.v1.Link.WorkflowEvent.RequestIdReference;

                            /**
                             * Creates a RequestIdReference message from a plain object. Also converts values to their respective internal types.
                             * @param object Plain object
                             * @returns RequestIdReference
                             */
                            public static fromObject(object: { [k: string]: any }): temporal.api.common.v1.Link.WorkflowEvent.RequestIdReference;

                            /**
                             * Creates a plain object from a RequestIdReference message. Also converts values to other types if specified.
                             * @param message RequestIdReference
                             * @param [options] Conversion options
                             * @returns Plain object
                             */
                            public static toObject(message: temporal.api.common.v1.Link.WorkflowEvent.RequestIdReference, options?: $protobuf.IConversionOptions): { [k: string]: any };

                            /**
                             * Converts this RequestIdReference to JSON.
                             * @returns JSON object
                             */
                            public toJSON(): { [k: string]: any };

                            /**
                             * Gets the default type url for RequestIdReference
                             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                             * @returns The default type url
                             */
                            public static getTypeUrl(typeUrlPrefix?: string): string;
                        }
                    }

                    /** Properties of a BatchJob. */
                    interface IBatchJob {

                        /** BatchJob jobId */
                        jobId?: (string|null);
                    }

                    /**
                     * A link to a built-in batch job.
                     * Batch jobs can be used to perform operations on a set of workflows (e.g. terminate, signal, cancel, etc).
                     * This link can be put on workflow history events generated by actions taken by a batch job.
                     */
                    class BatchJob implements IBatchJob {

                        /**
                         * Constructs a new BatchJob.
                         * @param [properties] Properties to set
                         */
                        constructor(properties?: temporal.api.common.v1.Link.IBatchJob);

                        /** BatchJob jobId. */
                        public jobId: string;

                        /**
                         * Creates a new BatchJob instance using the specified properties.
                         * @param [properties] Properties to set
                         * @returns BatchJob instance
                         */
                        public static create(properties?: temporal.api.common.v1.Link.IBatchJob): temporal.api.common.v1.Link.BatchJob;

                        /**
                         * Encodes the specified BatchJob message. Does not implicitly {@link temporal.api.common.v1.Link.BatchJob.verify|verify} messages.
                         * @param message BatchJob message or plain object to encode
                         * @param [writer] Writer to encode to
                         * @returns Writer
                         */
                        public static encode(message: temporal.api.common.v1.Link.IBatchJob, writer?: $protobuf.Writer): $protobuf.Writer;

                        /**
                         * Encodes the specified BatchJob message, length delimited. Does not implicitly {@link temporal.api.common.v1.Link.BatchJob.verify|verify} messages.
                         * @param message BatchJob message or plain object to encode
                         * @param [writer] Writer to encode to
                         * @returns Writer
                         */
                        public static encodeDelimited(message: temporal.api.common.v1.Link.IBatchJob, writer?: $protobuf.Writer): $protobuf.Writer;

                        /**
                         * Decodes a BatchJob message from the specified reader or buffer.
                         * @param reader Reader or buffer to decode from
                         * @param [length] Message length if known beforehand
                         * @returns BatchJob
                         * @throws {Error} If the payload is not a reader or valid buffer
                         * @throws {$protobuf.util.ProtocolError} If required fields are missing
                         */
                        public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.common.v1.Link.BatchJob;

                        /**
                         * Decodes a BatchJob message from the specified reader or buffer, length delimited.
                         * @param reader Reader or buffer to decode from
                         * @returns BatchJob
                         * @throws {Error} If the payload is not a reader or valid buffer
                         * @throws {$protobuf.util.ProtocolError} If required fields are missing
                         */
                        public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.common.v1.Link.BatchJob;

                        /**
                         * Creates a BatchJob message from a plain object. Also converts values to their respective internal types.
                         * @param object Plain object
                         * @returns BatchJob
                         */
                        public static fromObject(object: { [k: string]: any }): temporal.api.common.v1.Link.BatchJob;

                        /**
                         * Creates a plain object from a BatchJob message. Also converts values to other types if specified.
                         * @param message BatchJob
                         * @param [options] Conversion options
                         * @returns Plain object
                         */
                        public static toObject(message: temporal.api.common.v1.Link.BatchJob, options?: $protobuf.IConversionOptions): { [k: string]: any };

                        /**
                         * Converts this BatchJob to JSON.
                         * @returns JSON object
                         */
                        public toJSON(): { [k: string]: any };

                        /**
                         * Gets the default type url for BatchJob
                         * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                         * @returns The default type url
                         */
                        public static getTypeUrl(typeUrlPrefix?: string): string;
                    }
                }

                /** Properties of a Priority. */
                interface IPriority {

                    /**
                     * Priority key is a positive integer from 1 to n, where smaller integers
                     * correspond to higher priorities (tasks run sooner). In general, tasks in
                     * a queue should be processed in close to priority order, although small
                     * deviations are possible.
                     *
                     * The maximum priority value (minimum priority) is determined by server
                     * configuration, and defaults to 5.
                     *
                     * If priority is not present (or zero), then the effective priority will be
                     * the default priority, which is is calculated by (min+max)/2. With the
                     * default max of 5, and min of 1, that comes out to 3.
                     */
                    priorityKey?: (number|null);

                    /**
                     * Fairness key is a short string that's used as a key for a fairness
                     * balancing mechanism. It may correspond to a tenant id, or to a fixed
                     * string like "high" or "low". The default is the empty string.
                     *
                     * The fairness mechanism attempts to dispatch tasks for a given key in
                     * proportion to its weight. For example, using a thousand distinct tenant
                     * ids, each with a weight of 1.0 (the default) will result in each tenant
                     * getting a roughly equal share of task dispatch throughput.
                     *
                     * (Note: this does not imply equal share of worker capacity! Fairness
                     * decisions are made based on queue statistics, not
                     * current worker load.)
                     *
                     * As another example, using keys "high" and "low" with weight 9.0 and 1.0
                     * respectively will prefer dispatching "high" tasks over "low" tasks at a
                     * 9:1 ratio, while allowing either key to use all worker capacity if the
                     * other is not present.
                     *
                     * All fairness mechanisms, including rate limits, are best-effort and
                     * probabilistic. The results may not match what a "perfect" algorithm with
                     * infinite resources would produce. The more unique keys are used, the less
                     * accurate the results will be.
                     *
                     * Fairness keys are limited to 64 bytes.
                     */
                    fairnessKey?: (string|null);

                    /**
                     * Fairness weight for a task can come from multiple sources for
                     * flexibility. From highest to lowest precedence:
                     * 1. Weights for a small set of keys can be overridden in task queue
                     * configuration with an API.
                     * 2. It can be attached to the workflow/activity in this field.
                     * 3. The default weight of 1.0 will be used.
                     *
                     * Weight values are clamped to the range [0.001, 1000].
                     */
                    fairnessWeight?: (number|null);
                }

                /**
                 * Priority contains metadata that controls relative ordering of task processing
                 * when tasks are backed up in a queue. Initially, Priority will be used in
                 * matching (workflow and activity) task queues. Later it may be used in history
                 * task queues and in rate limiting decisions.
                 *
                 * Priority is attached to workflows and activities. By default, activities
                 * inherit Priority from the workflow that created them, but may override fields
                 * when an activity is started or modified.
                 *
                 * Despite being named "Priority", this message also contains fields that
                 * control "fairness" mechanisms.
                 *
                 * For all fields, the field not present or equal to zero/empty string means to
                 * inherit the value from the calling workflow, or if there is no calling
                 * workflow, then use the default value.
                 *
                 * For all fields other than fairness_key, the zero value isn't meaningful so
                 * there's no confusion between inherit/default and a meaningful value. For
                 * fairness_key, the empty string will be interpreted as "inherit". This means
                 * that if a workflow has a non-empty fairness key, you can't override the
                 * fairness key of its activity to the empty string.
                 *
                 * The overall semantics of Priority are:
                 * 1. First, consider "priority": higher priority (lower number) goes first.
                 * 2. Then, consider fairness: try to dispatch tasks for different fairness keys
                 * in proportion to their weight.
                 *
                 * Applications may use any subset of mechanisms that are useful to them and
                 * leave the other fields to use default values.
                 *
                 * Not all queues in the system may support the "full" semantics of all priority
                 * fields. (Currently only support in matching task queues is planned.)
                 */
                class Priority implements IPriority {

                    /**
                     * Constructs a new Priority.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.api.common.v1.IPriority);

                    /**
                     * Priority key is a positive integer from 1 to n, where smaller integers
                     * correspond to higher priorities (tasks run sooner). In general, tasks in
                     * a queue should be processed in close to priority order, although small
                     * deviations are possible.
                     *
                     * The maximum priority value (minimum priority) is determined by server
                     * configuration, and defaults to 5.
                     *
                     * If priority is not present (or zero), then the effective priority will be
                     * the default priority, which is is calculated by (min+max)/2. With the
                     * default max of 5, and min of 1, that comes out to 3.
                     */
                    public priorityKey: number;

                    /**
                     * Fairness key is a short string that's used as a key for a fairness
                     * balancing mechanism. It may correspond to a tenant id, or to a fixed
                     * string like "high" or "low". The default is the empty string.
                     *
                     * The fairness mechanism attempts to dispatch tasks for a given key in
                     * proportion to its weight. For example, using a thousand distinct tenant
                     * ids, each with a weight of 1.0 (the default) will result in each tenant
                     * getting a roughly equal share of task dispatch throughput.
                     *
                     * (Note: this does not imply equal share of worker capacity! Fairness
                     * decisions are made based on queue statistics, not
                     * current worker load.)
                     *
                     * As another example, using keys "high" and "low" with weight 9.0 and 1.0
                     * respectively will prefer dispatching "high" tasks over "low" tasks at a
                     * 9:1 ratio, while allowing either key to use all worker capacity if the
                     * other is not present.
                     *
                     * All fairness mechanisms, including rate limits, are best-effort and
                     * probabilistic. The results may not match what a "perfect" algorithm with
                     * infinite resources would produce. The more unique keys are used, the less
                     * accurate the results will be.
                     *
                     * Fairness keys are limited to 64 bytes.
                     */
                    public fairnessKey: string;

                    /**
                     * Fairness weight for a task can come from multiple sources for
                     * flexibility. From highest to lowest precedence:
                     * 1. Weights for a small set of keys can be overridden in task queue
                     * configuration with an API.
                     * 2. It can be attached to the workflow/activity in this field.
                     * 3. The default weight of 1.0 will be used.
                     *
                     * Weight values are clamped to the range [0.001, 1000].
                     */
                    public fairnessWeight: number;

                    /**
                     * Creates a new Priority instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns Priority instance
                     */
                    public static create(properties?: temporal.api.common.v1.IPriority): temporal.api.common.v1.Priority;

                    /**
                     * Encodes the specified Priority message. Does not implicitly {@link temporal.api.common.v1.Priority.verify|verify} messages.
                     * @param message Priority message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.api.common.v1.IPriority, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified Priority message, length delimited. Does not implicitly {@link temporal.api.common.v1.Priority.verify|verify} messages.
                     * @param message Priority message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.api.common.v1.IPriority, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a Priority message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns Priority
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.common.v1.Priority;

                    /**
                     * Decodes a Priority message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns Priority
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.common.v1.Priority;

                    /**
                     * Creates a Priority message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns Priority
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.api.common.v1.Priority;

                    /**
                     * Creates a plain object from a Priority message. Also converts values to other types if specified.
                     * @param message Priority
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.api.common.v1.Priority, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this Priority to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for Priority
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of a WorkerSelector. */
                interface IWorkerSelector {

                    /** Worker instance key to which the command should be sent. */
                    workerInstanceKey?: (string|null);
                }

                /**
                 * This is used to send commands to a specific worker or a group of workers.
                 * Right now, it is used to send commands to a specific worker instance.
                 * Will be extended to be able to send command to multiple workers.
                 */
                class WorkerSelector implements IWorkerSelector {

                    /**
                     * Constructs a new WorkerSelector.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.api.common.v1.IWorkerSelector);

                    /** Worker instance key to which the command should be sent. */
                    public workerInstanceKey?: (string|null);

                    /**
                     * Options are:
                     * - query (will be used as query to ListWorkers, same format as in ListWorkersRequest.query)
                     * - task queue (just a shortcut. Same as query=' "TaskQueue"="my-task-queue" ')
                     * - etc.
                     * All but 'query' are shortcuts, can be replaced with a query, but it is not convenient.
                     * string query = 5;
                     * string task_queue = 6;
                     * ...
                     */
                    public selector?: "workerInstanceKey";

                    /**
                     * Creates a new WorkerSelector instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns WorkerSelector instance
                     */
                    public static create(properties?: temporal.api.common.v1.IWorkerSelector): temporal.api.common.v1.WorkerSelector;

                    /**
                     * Encodes the specified WorkerSelector message. Does not implicitly {@link temporal.api.common.v1.WorkerSelector.verify|verify} messages.
                     * @param message WorkerSelector message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.api.common.v1.IWorkerSelector, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified WorkerSelector message, length delimited. Does not implicitly {@link temporal.api.common.v1.WorkerSelector.verify|verify} messages.
                     * @param message WorkerSelector message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.api.common.v1.IWorkerSelector, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a WorkerSelector message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns WorkerSelector
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.common.v1.WorkerSelector;

                    /**
                     * Decodes a WorkerSelector message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns WorkerSelector
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.common.v1.WorkerSelector;

                    /**
                     * Creates a WorkerSelector message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns WorkerSelector
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.api.common.v1.WorkerSelector;

                    /**
                     * Creates a plain object from a WorkerSelector message. Also converts values to other types if specified.
                     * @param message WorkerSelector
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.api.common.v1.WorkerSelector, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this WorkerSelector to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for WorkerSelector
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }
            }
        }

        /** Namespace enums. */
        namespace enums {

            /** Namespace v1. */
            namespace v1 {

                /** EncodingType enum. */
                enum EncodingType {
                    ENCODING_TYPE_UNSPECIFIED = 0,
                    ENCODING_TYPE_PROTO3 = 1,
                    ENCODING_TYPE_JSON = 2
                }

                /** IndexedValueType enum. */
                enum IndexedValueType {
                    INDEXED_VALUE_TYPE_UNSPECIFIED = 0,
                    INDEXED_VALUE_TYPE_TEXT = 1,
                    INDEXED_VALUE_TYPE_KEYWORD = 2,
                    INDEXED_VALUE_TYPE_INT = 3,
                    INDEXED_VALUE_TYPE_DOUBLE = 4,
                    INDEXED_VALUE_TYPE_BOOL = 5,
                    INDEXED_VALUE_TYPE_DATETIME = 6,
                    INDEXED_VALUE_TYPE_KEYWORD_LIST = 7
                }

                /** Severity enum. */
                enum Severity {
                    SEVERITY_UNSPECIFIED = 0,
                    SEVERITY_HIGH = 1,
                    SEVERITY_MEDIUM = 2,
                    SEVERITY_LOW = 3
                }

                /** State of a callback. */
                enum CallbackState {
                    CALLBACK_STATE_UNSPECIFIED = 0,
                    CALLBACK_STATE_STANDBY = 1,
                    CALLBACK_STATE_SCHEDULED = 2,
                    CALLBACK_STATE_BACKING_OFF = 3,
                    CALLBACK_STATE_FAILED = 4,
                    CALLBACK_STATE_SUCCEEDED = 5,
                    CALLBACK_STATE_BLOCKED = 6
                }

                /** State of a pending Nexus operation. */
                enum PendingNexusOperationState {
                    PENDING_NEXUS_OPERATION_STATE_UNSPECIFIED = 0,
                    PENDING_NEXUS_OPERATION_STATE_SCHEDULED = 1,
                    PENDING_NEXUS_OPERATION_STATE_BACKING_OFF = 2,
                    PENDING_NEXUS_OPERATION_STATE_STARTED = 3,
                    PENDING_NEXUS_OPERATION_STATE_BLOCKED = 4
                }

                /** State of a Nexus operation cancellation. */
                enum NexusOperationCancellationState {
                    NEXUS_OPERATION_CANCELLATION_STATE_UNSPECIFIED = 0,
                    NEXUS_OPERATION_CANCELLATION_STATE_SCHEDULED = 1,
                    NEXUS_OPERATION_CANCELLATION_STATE_BACKING_OFF = 2,
                    NEXUS_OPERATION_CANCELLATION_STATE_SUCCEEDED = 3,
                    NEXUS_OPERATION_CANCELLATION_STATE_FAILED = 4,
                    NEXUS_OPERATION_CANCELLATION_STATE_TIMED_OUT = 5,
                    NEXUS_OPERATION_CANCELLATION_STATE_BLOCKED = 6
                }

                /** WorkflowRuleActionScope enum. */
                enum WorkflowRuleActionScope {
                    WORKFLOW_RULE_ACTION_SCOPE_UNSPECIFIED = 0,
                    WORKFLOW_RULE_ACTION_SCOPE_WORKFLOW = 1,
                    WORKFLOW_RULE_ACTION_SCOPE_ACTIVITY = 2
                }

                /** ApplicationErrorCategory enum. */
                enum ApplicationErrorCategory {
                    APPLICATION_ERROR_CATEGORY_UNSPECIFIED = 0,
                    APPLICATION_ERROR_CATEGORY_BENIGN = 1
                }

                /**
                 * (-- api-linter: core::0216::synonyms=disabled
                 * aip.dev/not-precedent: It seems we have both state and status, and status is a better fit for workers. --)
                 */
                enum WorkerStatus {
                    WORKER_STATUS_UNSPECIFIED = 0,
                    WORKER_STATUS_RUNNING = 1,
                    WORKER_STATUS_SHUTTING_DOWN = 2,
                    WORKER_STATUS_SHUTDOWN = 3
                }

                /** Whenever this list of events is changed do change the function shouldBufferEvent in mutableStateBuilder.go to make sure to do the correct event ordering */
                enum EventType {
                    EVENT_TYPE_UNSPECIFIED = 0,
                    EVENT_TYPE_WORKFLOW_EXECUTION_STARTED = 1,
                    EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED = 2,
                    EVENT_TYPE_WORKFLOW_EXECUTION_FAILED = 3,
                    EVENT_TYPE_WORKFLOW_EXECUTION_TIMED_OUT = 4,
                    EVENT_TYPE_WORKFLOW_TASK_SCHEDULED = 5,
                    EVENT_TYPE_WORKFLOW_TASK_STARTED = 6,
                    EVENT_TYPE_WORKFLOW_TASK_COMPLETED = 7,
                    EVENT_TYPE_WORKFLOW_TASK_TIMED_OUT = 8,
                    EVENT_TYPE_WORKFLOW_TASK_FAILED = 9,
                    EVENT_TYPE_ACTIVITY_TASK_SCHEDULED = 10,
                    EVENT_TYPE_ACTIVITY_TASK_STARTED = 11,
                    EVENT_TYPE_ACTIVITY_TASK_COMPLETED = 12,
                    EVENT_TYPE_ACTIVITY_TASK_FAILED = 13,
                    EVENT_TYPE_ACTIVITY_TASK_TIMED_OUT = 14,
                    EVENT_TYPE_ACTIVITY_TASK_CANCEL_REQUESTED = 15,
                    EVENT_TYPE_ACTIVITY_TASK_CANCELED = 16,
                    EVENT_TYPE_TIMER_STARTED = 17,
                    EVENT_TYPE_TIMER_FIRED = 18,
                    EVENT_TYPE_TIMER_CANCELED = 19,
                    EVENT_TYPE_WORKFLOW_EXECUTION_CANCEL_REQUESTED = 20,
                    EVENT_TYPE_WORKFLOW_EXECUTION_CANCELED = 21,
                    EVENT_TYPE_REQUEST_CANCEL_EXTERNAL_WORKFLOW_EXECUTION_INITIATED = 22,
                    EVENT_TYPE_REQUEST_CANCEL_EXTERNAL_WORKFLOW_EXECUTION_FAILED = 23,
                    EVENT_TYPE_EXTERNAL_WORKFLOW_EXECUTION_CANCEL_REQUESTED = 24,
                    EVENT_TYPE_MARKER_RECORDED = 25,
                    EVENT_TYPE_WORKFLOW_EXECUTION_SIGNALED = 26,
                    EVENT_TYPE_WORKFLOW_EXECUTION_TERMINATED = 27,
                    EVENT_TYPE_WORKFLOW_EXECUTION_CONTINUED_AS_NEW = 28,
                    EVENT_TYPE_START_CHILD_WORKFLOW_EXECUTION_INITIATED = 29,
                    EVENT_TYPE_START_CHILD_WORKFLOW_EXECUTION_FAILED = 30,
                    EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_STARTED = 31,
                    EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_COMPLETED = 32,
                    EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_FAILED = 33,
                    EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_CANCELED = 34,
                    EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_TIMED_OUT = 35,
                    EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_TERMINATED = 36,
                    EVENT_TYPE_SIGNAL_EXTERNAL_WORKFLOW_EXECUTION_INITIATED = 37,
                    EVENT_TYPE_SIGNAL_EXTERNAL_WORKFLOW_EXECUTION_FAILED = 38,
                    EVENT_TYPE_EXTERNAL_WORKFLOW_EXECUTION_SIGNALED = 39,
                    EVENT_TYPE_UPSERT_WORKFLOW_SEARCH_ATTRIBUTES = 40,
                    EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_ADMITTED = 47,
                    EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_ACCEPTED = 41,
                    EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_REJECTED = 42,
                    EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_COMPLETED = 43,
                    EVENT_TYPE_WORKFLOW_PROPERTIES_MODIFIED_EXTERNALLY = 44,
                    EVENT_TYPE_ACTIVITY_PROPERTIES_MODIFIED_EXTERNALLY = 45,
                    EVENT_TYPE_WORKFLOW_PROPERTIES_MODIFIED = 46,
                    EVENT_TYPE_NEXUS_OPERATION_SCHEDULED = 48,
                    EVENT_TYPE_NEXUS_OPERATION_STARTED = 49,
                    EVENT_TYPE_NEXUS_OPERATION_COMPLETED = 50,
                    EVENT_TYPE_NEXUS_OPERATION_FAILED = 51,
                    EVENT_TYPE_NEXUS_OPERATION_CANCELED = 52,
                    EVENT_TYPE_NEXUS_OPERATION_TIMED_OUT = 53,
                    EVENT_TYPE_NEXUS_OPERATION_CANCEL_REQUESTED = 54,
                    EVENT_TYPE_WORKFLOW_EXECUTION_OPTIONS_UPDATED = 55,
                    EVENT_TYPE_NEXUS_OPERATION_CANCEL_REQUEST_COMPLETED = 56,
                    EVENT_TYPE_NEXUS_OPERATION_CANCEL_REQUEST_FAILED = 57
                }

                /** Event types to exclude when reapplying events beyond the reset point. */
                enum ResetReapplyExcludeType {
                    RESET_REAPPLY_EXCLUDE_TYPE_UNSPECIFIED = 0,
                    RESET_REAPPLY_EXCLUDE_TYPE_SIGNAL = 1,
                    RESET_REAPPLY_EXCLUDE_TYPE_UPDATE = 2,
                    RESET_REAPPLY_EXCLUDE_TYPE_NEXUS = 3,
                    RESET_REAPPLY_EXCLUDE_TYPE_CANCEL_REQUEST = 4
                }

                /**
                 * Deprecated: applications should use ResetReapplyExcludeType to specify
                 * exclusions from this set, and new event types should be added to ResetReapplyExcludeType
                 * instead of here.
                 */
                enum ResetReapplyType {
                    RESET_REAPPLY_TYPE_UNSPECIFIED = 0,
                    RESET_REAPPLY_TYPE_SIGNAL = 1,
                    RESET_REAPPLY_TYPE_NONE = 2,
                    RESET_REAPPLY_TYPE_ALL_ELIGIBLE = 3
                }

                /** Deprecated, see temporal.api.common.v1.ResetOptions. */
                enum ResetType {
                    RESET_TYPE_UNSPECIFIED = 0,
                    RESET_TYPE_FIRST_WORKFLOW_TASK = 1,
                    RESET_TYPE_LAST_WORKFLOW_TASK = 2
                }

                /**
                 * Defines whether to allow re-using a workflow id from a previously *closed* workflow.
                 * If the request is denied, the server returns a `WorkflowExecutionAlreadyStartedFailure` error.
                 *
                 * See `WorkflowIdConflictPolicy` for handling workflow id duplication with a *running* workflow.
                 */
                enum WorkflowIdReusePolicy {
                    WORKFLOW_ID_REUSE_POLICY_UNSPECIFIED = 0,
                    WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE = 1,
                    WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY = 2,
                    WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE = 3,
                    WORKFLOW_ID_REUSE_POLICY_TERMINATE_IF_RUNNING = 4
                }

                /**
                 * Defines what to do when trying to start a workflow with the same workflow id as a *running* workflow.
                 * Note that it is *never* valid to have two actively running instances of the same workflow id.
                 *
                 * See `WorkflowIdReusePolicy` for handling workflow id duplication with a *closed* workflow.
                 */
                enum WorkflowIdConflictPolicy {
                    WORKFLOW_ID_CONFLICT_POLICY_UNSPECIFIED = 0,
                    WORKFLOW_ID_CONFLICT_POLICY_FAIL = 1,
                    WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING = 2,
                    WORKFLOW_ID_CONFLICT_POLICY_TERMINATE_EXISTING = 3
                }

                /** Defines how child workflows will react to their parent completing */
                enum ParentClosePolicy {
                    PARENT_CLOSE_POLICY_UNSPECIFIED = 0,
                    PARENT_CLOSE_POLICY_TERMINATE = 1,
                    PARENT_CLOSE_POLICY_ABANDON = 2,
                    PARENT_CLOSE_POLICY_REQUEST_CANCEL = 3
                }

                /** ContinueAsNewInitiator enum. */
                enum ContinueAsNewInitiator {
                    CONTINUE_AS_NEW_INITIATOR_UNSPECIFIED = 0,
                    CONTINUE_AS_NEW_INITIATOR_WORKFLOW = 1,
                    CONTINUE_AS_NEW_INITIATOR_RETRY = 2,
                    CONTINUE_AS_NEW_INITIATOR_CRON_SCHEDULE = 3
                }

                /**
                 * (-- api-linter: core::0216::synonyms=disabled
                 * aip.dev/not-precedent: There is WorkflowExecutionState already in another package. --)
                 */
                enum WorkflowExecutionStatus {
                    WORKFLOW_EXECUTION_STATUS_UNSPECIFIED = 0,
                    WORKFLOW_EXECUTION_STATUS_RUNNING = 1,
                    WORKFLOW_EXECUTION_STATUS_COMPLETED = 2,
                    WORKFLOW_EXECUTION_STATUS_FAILED = 3,
                    WORKFLOW_EXECUTION_STATUS_CANCELED = 4,
                    WORKFLOW_EXECUTION_STATUS_TERMINATED = 5,
                    WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW = 6,
                    WORKFLOW_EXECUTION_STATUS_TIMED_OUT = 7
                }

                /** PendingActivityState enum. */
                enum PendingActivityState {
                    PENDING_ACTIVITY_STATE_UNSPECIFIED = 0,
                    PENDING_ACTIVITY_STATE_SCHEDULED = 1,
                    PENDING_ACTIVITY_STATE_STARTED = 2,
                    PENDING_ACTIVITY_STATE_CANCEL_REQUESTED = 3,
                    PENDING_ACTIVITY_STATE_PAUSED = 4,
                    PENDING_ACTIVITY_STATE_PAUSE_REQUESTED = 5
                }

                /** PendingWorkflowTaskState enum. */
                enum PendingWorkflowTaskState {
                    PENDING_WORKFLOW_TASK_STATE_UNSPECIFIED = 0,
                    PENDING_WORKFLOW_TASK_STATE_SCHEDULED = 1,
                    PENDING_WORKFLOW_TASK_STATE_STARTED = 2
                }

                /** HistoryEventFilterType enum. */
                enum HistoryEventFilterType {
                    HISTORY_EVENT_FILTER_TYPE_UNSPECIFIED = 0,
                    HISTORY_EVENT_FILTER_TYPE_ALL_EVENT = 1,
                    HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT = 2
                }

                /** RetryState enum. */
                enum RetryState {
                    RETRY_STATE_UNSPECIFIED = 0,
                    RETRY_STATE_IN_PROGRESS = 1,
                    RETRY_STATE_NON_RETRYABLE_FAILURE = 2,
                    RETRY_STATE_TIMEOUT = 3,
                    RETRY_STATE_MAXIMUM_ATTEMPTS_REACHED = 4,
                    RETRY_STATE_RETRY_POLICY_NOT_SET = 5,
                    RETRY_STATE_INTERNAL_SERVER_ERROR = 6,
                    RETRY_STATE_CANCEL_REQUESTED = 7
                }

                /** TimeoutType enum. */
                enum TimeoutType {
                    TIMEOUT_TYPE_UNSPECIFIED = 0,
                    TIMEOUT_TYPE_START_TO_CLOSE = 1,
                    TIMEOUT_TYPE_SCHEDULE_TO_START = 2,
                    TIMEOUT_TYPE_SCHEDULE_TO_CLOSE = 3,
                    TIMEOUT_TYPE_HEARTBEAT = 4
                }

                /**
                 * Versioning Behavior specifies if and how a workflow execution moves between Worker Deployment
                 * Versions. The Versioning Behavior of a workflow execution is typically specified by the worker
                 * who completes the first task of the execution, but is also overridable manually for new and
                 * existing workflows (see VersioningOverride).
                 * Experimental. Worker Deployments are experimental and might significantly change in the future.
                 */
                enum VersioningBehavior {
                    VERSIONING_BEHAVIOR_UNSPECIFIED = 0,
                    VERSIONING_BEHAVIOR_PINNED = 1,
                    VERSIONING_BEHAVIOR_AUTO_UPGRADE = 2
                }

                /**
                 * NexusHandlerErrorRetryBehavior allows nexus handlers to explicity set the retry behavior of a HandlerError. If not
                 * specified, retry behavior is determined from the error type. For example internal errors are not retryable by default
                 * unless specified otherwise.
                 */
                enum NexusHandlerErrorRetryBehavior {
                    NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_UNSPECIFIED = 0,
                    NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_RETRYABLE = 1,
                    NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_NON_RETRYABLE = 2
                }
            }
        }

        /** Namespace failure. */
        namespace failure {

            /** Namespace v1. */
            namespace v1 {

                /** Properties of an ApplicationFailureInfo. */
                interface IApplicationFailureInfo {

                    /** ApplicationFailureInfo type */
                    type?: (string|null);

                    /** ApplicationFailureInfo nonRetryable */
                    nonRetryable?: (boolean|null);

                    /** ApplicationFailureInfo details */
                    details?: (temporal.api.common.v1.IPayloads|null);

                    /**
                     * next_retry_delay can be used by the client to override the activity
                     * retry interval calculated by the retry policy. Retry attempts will
                     * still be subject to the maximum retries limit and total time limit
                     * defined by the policy.
                     */
                    nextRetryDelay?: (google.protobuf.IDuration|null);

                    /** ApplicationFailureInfo category */
                    category?: (temporal.api.enums.v1.ApplicationErrorCategory|null);
                }

                /** Represents an ApplicationFailureInfo. */
                class ApplicationFailureInfo implements IApplicationFailureInfo {

                    /**
                     * Constructs a new ApplicationFailureInfo.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.api.failure.v1.IApplicationFailureInfo);

                    /** ApplicationFailureInfo type. */
                    public type: string;

                    /** ApplicationFailureInfo nonRetryable. */
                    public nonRetryable: boolean;

                    /** ApplicationFailureInfo details. */
                    public details?: (temporal.api.common.v1.IPayloads|null);

                    /**
                     * next_retry_delay can be used by the client to override the activity
                     * retry interval calculated by the retry policy. Retry attempts will
                     * still be subject to the maximum retries limit and total time limit
                     * defined by the policy.
                     */
                    public nextRetryDelay?: (google.protobuf.IDuration|null);

                    /** ApplicationFailureInfo category. */
                    public category: temporal.api.enums.v1.ApplicationErrorCategory;

                    /**
                     * Creates a new ApplicationFailureInfo instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns ApplicationFailureInfo instance
                     */
                    public static create(properties?: temporal.api.failure.v1.IApplicationFailureInfo): temporal.api.failure.v1.ApplicationFailureInfo;

                    /**
                     * Encodes the specified ApplicationFailureInfo message. Does not implicitly {@link temporal.api.failure.v1.ApplicationFailureInfo.verify|verify} messages.
                     * @param message ApplicationFailureInfo message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.api.failure.v1.IApplicationFailureInfo, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified ApplicationFailureInfo message, length delimited. Does not implicitly {@link temporal.api.failure.v1.ApplicationFailureInfo.verify|verify} messages.
                     * @param message ApplicationFailureInfo message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.api.failure.v1.IApplicationFailureInfo, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes an ApplicationFailureInfo message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns ApplicationFailureInfo
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.failure.v1.ApplicationFailureInfo;

                    /**
                     * Decodes an ApplicationFailureInfo message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns ApplicationFailureInfo
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.failure.v1.ApplicationFailureInfo;

                    /**
                     * Creates an ApplicationFailureInfo message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns ApplicationFailureInfo
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.api.failure.v1.ApplicationFailureInfo;

                    /**
                     * Creates a plain object from an ApplicationFailureInfo message. Also converts values to other types if specified.
                     * @param message ApplicationFailureInfo
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.api.failure.v1.ApplicationFailureInfo, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this ApplicationFailureInfo to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for ApplicationFailureInfo
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of a TimeoutFailureInfo. */
                interface ITimeoutFailureInfo {

                    /** TimeoutFailureInfo timeoutType */
                    timeoutType?: (temporal.api.enums.v1.TimeoutType|null);

                    /** TimeoutFailureInfo lastHeartbeatDetails */
                    lastHeartbeatDetails?: (temporal.api.common.v1.IPayloads|null);
                }

                /** Represents a TimeoutFailureInfo. */
                class TimeoutFailureInfo implements ITimeoutFailureInfo {

                    /**
                     * Constructs a new TimeoutFailureInfo.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.api.failure.v1.ITimeoutFailureInfo);

                    /** TimeoutFailureInfo timeoutType. */
                    public timeoutType: temporal.api.enums.v1.TimeoutType;

                    /** TimeoutFailureInfo lastHeartbeatDetails. */
                    public lastHeartbeatDetails?: (temporal.api.common.v1.IPayloads|null);

                    /**
                     * Creates a new TimeoutFailureInfo instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns TimeoutFailureInfo instance
                     */
                    public static create(properties?: temporal.api.failure.v1.ITimeoutFailureInfo): temporal.api.failure.v1.TimeoutFailureInfo;

                    /**
                     * Encodes the specified TimeoutFailureInfo message. Does not implicitly {@link temporal.api.failure.v1.TimeoutFailureInfo.verify|verify} messages.
                     * @param message TimeoutFailureInfo message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.api.failure.v1.ITimeoutFailureInfo, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified TimeoutFailureInfo message, length delimited. Does not implicitly {@link temporal.api.failure.v1.TimeoutFailureInfo.verify|verify} messages.
                     * @param message TimeoutFailureInfo message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.api.failure.v1.ITimeoutFailureInfo, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a TimeoutFailureInfo message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns TimeoutFailureInfo
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.failure.v1.TimeoutFailureInfo;

                    /**
                     * Decodes a TimeoutFailureInfo message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns TimeoutFailureInfo
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.failure.v1.TimeoutFailureInfo;

                    /**
                     * Creates a TimeoutFailureInfo message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns TimeoutFailureInfo
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.api.failure.v1.TimeoutFailureInfo;

                    /**
                     * Creates a plain object from a TimeoutFailureInfo message. Also converts values to other types if specified.
                     * @param message TimeoutFailureInfo
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.api.failure.v1.TimeoutFailureInfo, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this TimeoutFailureInfo to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for TimeoutFailureInfo
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of a CanceledFailureInfo. */
                interface ICanceledFailureInfo {

                    /** CanceledFailureInfo details */
                    details?: (temporal.api.common.v1.IPayloads|null);
                }

                /** Represents a CanceledFailureInfo. */
                class CanceledFailureInfo implements ICanceledFailureInfo {

                    /**
                     * Constructs a new CanceledFailureInfo.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.api.failure.v1.ICanceledFailureInfo);

                    /** CanceledFailureInfo details. */
                    public details?: (temporal.api.common.v1.IPayloads|null);

                    /**
                     * Creates a new CanceledFailureInfo instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns CanceledFailureInfo instance
                     */
                    public static create(properties?: temporal.api.failure.v1.ICanceledFailureInfo): temporal.api.failure.v1.CanceledFailureInfo;

                    /**
                     * Encodes the specified CanceledFailureInfo message. Does not implicitly {@link temporal.api.failure.v1.CanceledFailureInfo.verify|verify} messages.
                     * @param message CanceledFailureInfo message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.api.failure.v1.ICanceledFailureInfo, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified CanceledFailureInfo message, length delimited. Does not implicitly {@link temporal.api.failure.v1.CanceledFailureInfo.verify|verify} messages.
                     * @param message CanceledFailureInfo message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.api.failure.v1.ICanceledFailureInfo, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a CanceledFailureInfo message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns CanceledFailureInfo
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.failure.v1.CanceledFailureInfo;

                    /**
                     * Decodes a CanceledFailureInfo message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns CanceledFailureInfo
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.failure.v1.CanceledFailureInfo;

                    /**
                     * Creates a CanceledFailureInfo message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns CanceledFailureInfo
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.api.failure.v1.CanceledFailureInfo;

                    /**
                     * Creates a plain object from a CanceledFailureInfo message. Also converts values to other types if specified.
                     * @param message CanceledFailureInfo
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.api.failure.v1.CanceledFailureInfo, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this CanceledFailureInfo to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for CanceledFailureInfo
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of a TerminatedFailureInfo. */
                interface ITerminatedFailureInfo {
                }

                /** Represents a TerminatedFailureInfo. */
                class TerminatedFailureInfo implements ITerminatedFailureInfo {

                    /**
                     * Constructs a new TerminatedFailureInfo.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.api.failure.v1.ITerminatedFailureInfo);

                    /**
                     * Creates a new TerminatedFailureInfo instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns TerminatedFailureInfo instance
                     */
                    public static create(properties?: temporal.api.failure.v1.ITerminatedFailureInfo): temporal.api.failure.v1.TerminatedFailureInfo;

                    /**
                     * Encodes the specified TerminatedFailureInfo message. Does not implicitly {@link temporal.api.failure.v1.TerminatedFailureInfo.verify|verify} messages.
                     * @param message TerminatedFailureInfo message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.api.failure.v1.ITerminatedFailureInfo, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified TerminatedFailureInfo message, length delimited. Does not implicitly {@link temporal.api.failure.v1.TerminatedFailureInfo.verify|verify} messages.
                     * @param message TerminatedFailureInfo message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.api.failure.v1.ITerminatedFailureInfo, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a TerminatedFailureInfo message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns TerminatedFailureInfo
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.failure.v1.TerminatedFailureInfo;

                    /**
                     * Decodes a TerminatedFailureInfo message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns TerminatedFailureInfo
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.failure.v1.TerminatedFailureInfo;

                    /**
                     * Creates a TerminatedFailureInfo message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns TerminatedFailureInfo
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.api.failure.v1.TerminatedFailureInfo;

                    /**
                     * Creates a plain object from a TerminatedFailureInfo message. Also converts values to other types if specified.
                     * @param message TerminatedFailureInfo
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.api.failure.v1.TerminatedFailureInfo, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this TerminatedFailureInfo to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for TerminatedFailureInfo
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of a ServerFailureInfo. */
                interface IServerFailureInfo {

                    /** ServerFailureInfo nonRetryable */
                    nonRetryable?: (boolean|null);
                }

                /** Represents a ServerFailureInfo. */
                class ServerFailureInfo implements IServerFailureInfo {

                    /**
                     * Constructs a new ServerFailureInfo.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.api.failure.v1.IServerFailureInfo);

                    /** ServerFailureInfo nonRetryable. */
                    public nonRetryable: boolean;

                    /**
                     * Creates a new ServerFailureInfo instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns ServerFailureInfo instance
                     */
                    public static create(properties?: temporal.api.failure.v1.IServerFailureInfo): temporal.api.failure.v1.ServerFailureInfo;

                    /**
                     * Encodes the specified ServerFailureInfo message. Does not implicitly {@link temporal.api.failure.v1.ServerFailureInfo.verify|verify} messages.
                     * @param message ServerFailureInfo message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.api.failure.v1.IServerFailureInfo, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified ServerFailureInfo message, length delimited. Does not implicitly {@link temporal.api.failure.v1.ServerFailureInfo.verify|verify} messages.
                     * @param message ServerFailureInfo message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.api.failure.v1.IServerFailureInfo, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a ServerFailureInfo message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns ServerFailureInfo
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.failure.v1.ServerFailureInfo;

                    /**
                     * Decodes a ServerFailureInfo message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns ServerFailureInfo
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.failure.v1.ServerFailureInfo;

                    /**
                     * Creates a ServerFailureInfo message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns ServerFailureInfo
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.api.failure.v1.ServerFailureInfo;

                    /**
                     * Creates a plain object from a ServerFailureInfo message. Also converts values to other types if specified.
                     * @param message ServerFailureInfo
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.api.failure.v1.ServerFailureInfo, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this ServerFailureInfo to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for ServerFailureInfo
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of a ResetWorkflowFailureInfo. */
                interface IResetWorkflowFailureInfo {

                    /** ResetWorkflowFailureInfo lastHeartbeatDetails */
                    lastHeartbeatDetails?: (temporal.api.common.v1.IPayloads|null);
                }

                /** Represents a ResetWorkflowFailureInfo. */
                class ResetWorkflowFailureInfo implements IResetWorkflowFailureInfo {

                    /**
                     * Constructs a new ResetWorkflowFailureInfo.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.api.failure.v1.IResetWorkflowFailureInfo);

                    /** ResetWorkflowFailureInfo lastHeartbeatDetails. */
                    public lastHeartbeatDetails?: (temporal.api.common.v1.IPayloads|null);

                    /**
                     * Creates a new ResetWorkflowFailureInfo instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns ResetWorkflowFailureInfo instance
                     */
                    public static create(properties?: temporal.api.failure.v1.IResetWorkflowFailureInfo): temporal.api.failure.v1.ResetWorkflowFailureInfo;

                    /**
                     * Encodes the specified ResetWorkflowFailureInfo message. Does not implicitly {@link temporal.api.failure.v1.ResetWorkflowFailureInfo.verify|verify} messages.
                     * @param message ResetWorkflowFailureInfo message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.api.failure.v1.IResetWorkflowFailureInfo, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified ResetWorkflowFailureInfo message, length delimited. Does not implicitly {@link temporal.api.failure.v1.ResetWorkflowFailureInfo.verify|verify} messages.
                     * @param message ResetWorkflowFailureInfo message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.api.failure.v1.IResetWorkflowFailureInfo, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a ResetWorkflowFailureInfo message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns ResetWorkflowFailureInfo
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.failure.v1.ResetWorkflowFailureInfo;

                    /**
                     * Decodes a ResetWorkflowFailureInfo message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns ResetWorkflowFailureInfo
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.failure.v1.ResetWorkflowFailureInfo;

                    /**
                     * Creates a ResetWorkflowFailureInfo message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns ResetWorkflowFailureInfo
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.api.failure.v1.ResetWorkflowFailureInfo;

                    /**
                     * Creates a plain object from a ResetWorkflowFailureInfo message. Also converts values to other types if specified.
                     * @param message ResetWorkflowFailureInfo
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.api.failure.v1.ResetWorkflowFailureInfo, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this ResetWorkflowFailureInfo to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for ResetWorkflowFailureInfo
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of an ActivityFailureInfo. */
                interface IActivityFailureInfo {

                    /** ActivityFailureInfo scheduledEventId */
                    scheduledEventId?: (Long|null);

                    /** ActivityFailureInfo startedEventId */
                    startedEventId?: (Long|null);

                    /** ActivityFailureInfo identity */
                    identity?: (string|null);

                    /** ActivityFailureInfo activityType */
                    activityType?: (temporal.api.common.v1.IActivityType|null);

                    /** ActivityFailureInfo activityId */
                    activityId?: (string|null);

                    /** ActivityFailureInfo retryState */
                    retryState?: (temporal.api.enums.v1.RetryState|null);
                }

                /** Represents an ActivityFailureInfo. */
                class ActivityFailureInfo implements IActivityFailureInfo {

                    /**
                     * Constructs a new ActivityFailureInfo.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.api.failure.v1.IActivityFailureInfo);

                    /** ActivityFailureInfo scheduledEventId. */
                    public scheduledEventId: Long;

                    /** ActivityFailureInfo startedEventId. */
                    public startedEventId: Long;

                    /** ActivityFailureInfo identity. */
                    public identity: string;

                    /** ActivityFailureInfo activityType. */
                    public activityType?: (temporal.api.common.v1.IActivityType|null);

                    /** ActivityFailureInfo activityId. */
                    public activityId: string;

                    /** ActivityFailureInfo retryState. */
                    public retryState: temporal.api.enums.v1.RetryState;

                    /**
                     * Creates a new ActivityFailureInfo instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns ActivityFailureInfo instance
                     */
                    public static create(properties?: temporal.api.failure.v1.IActivityFailureInfo): temporal.api.failure.v1.ActivityFailureInfo;

                    /**
                     * Encodes the specified ActivityFailureInfo message. Does not implicitly {@link temporal.api.failure.v1.ActivityFailureInfo.verify|verify} messages.
                     * @param message ActivityFailureInfo message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.api.failure.v1.IActivityFailureInfo, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified ActivityFailureInfo message, length delimited. Does not implicitly {@link temporal.api.failure.v1.ActivityFailureInfo.verify|verify} messages.
                     * @param message ActivityFailureInfo message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.api.failure.v1.IActivityFailureInfo, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes an ActivityFailureInfo message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns ActivityFailureInfo
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.failure.v1.ActivityFailureInfo;

                    /**
                     * Decodes an ActivityFailureInfo message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns ActivityFailureInfo
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.failure.v1.ActivityFailureInfo;

                    /**
                     * Creates an ActivityFailureInfo message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns ActivityFailureInfo
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.api.failure.v1.ActivityFailureInfo;

                    /**
                     * Creates a plain object from an ActivityFailureInfo message. Also converts values to other types if specified.
                     * @param message ActivityFailureInfo
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.api.failure.v1.ActivityFailureInfo, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this ActivityFailureInfo to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for ActivityFailureInfo
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of a ChildWorkflowExecutionFailureInfo. */
                interface IChildWorkflowExecutionFailureInfo {

                    /** ChildWorkflowExecutionFailureInfo namespace */
                    namespace?: (string|null);

                    /** ChildWorkflowExecutionFailureInfo workflowExecution */
                    workflowExecution?: (temporal.api.common.v1.IWorkflowExecution|null);

                    /** ChildWorkflowExecutionFailureInfo workflowType */
                    workflowType?: (temporal.api.common.v1.IWorkflowType|null);

                    /** ChildWorkflowExecutionFailureInfo initiatedEventId */
                    initiatedEventId?: (Long|null);

                    /** ChildWorkflowExecutionFailureInfo startedEventId */
                    startedEventId?: (Long|null);

                    /** ChildWorkflowExecutionFailureInfo retryState */
                    retryState?: (temporal.api.enums.v1.RetryState|null);
                }

                /** Represents a ChildWorkflowExecutionFailureInfo. */
                class ChildWorkflowExecutionFailureInfo implements IChildWorkflowExecutionFailureInfo {

                    /**
                     * Constructs a new ChildWorkflowExecutionFailureInfo.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.api.failure.v1.IChildWorkflowExecutionFailureInfo);

                    /** ChildWorkflowExecutionFailureInfo namespace. */
                    public namespace: string;

                    /** ChildWorkflowExecutionFailureInfo workflowExecution. */
                    public workflowExecution?: (temporal.api.common.v1.IWorkflowExecution|null);

                    /** ChildWorkflowExecutionFailureInfo workflowType. */
                    public workflowType?: (temporal.api.common.v1.IWorkflowType|null);

                    /** ChildWorkflowExecutionFailureInfo initiatedEventId. */
                    public initiatedEventId: Long;

                    /** ChildWorkflowExecutionFailureInfo startedEventId. */
                    public startedEventId: Long;

                    /** ChildWorkflowExecutionFailureInfo retryState. */
                    public retryState: temporal.api.enums.v1.RetryState;

                    /**
                     * Creates a new ChildWorkflowExecutionFailureInfo instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns ChildWorkflowExecutionFailureInfo instance
                     */
                    public static create(properties?: temporal.api.failure.v1.IChildWorkflowExecutionFailureInfo): temporal.api.failure.v1.ChildWorkflowExecutionFailureInfo;

                    /**
                     * Encodes the specified ChildWorkflowExecutionFailureInfo message. Does not implicitly {@link temporal.api.failure.v1.ChildWorkflowExecutionFailureInfo.verify|verify} messages.
                     * @param message ChildWorkflowExecutionFailureInfo message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.api.failure.v1.IChildWorkflowExecutionFailureInfo, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified ChildWorkflowExecutionFailureInfo message, length delimited. Does not implicitly {@link temporal.api.failure.v1.ChildWorkflowExecutionFailureInfo.verify|verify} messages.
                     * @param message ChildWorkflowExecutionFailureInfo message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.api.failure.v1.IChildWorkflowExecutionFailureInfo, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a ChildWorkflowExecutionFailureInfo message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns ChildWorkflowExecutionFailureInfo
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.failure.v1.ChildWorkflowExecutionFailureInfo;

                    /**
                     * Decodes a ChildWorkflowExecutionFailureInfo message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns ChildWorkflowExecutionFailureInfo
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.failure.v1.ChildWorkflowExecutionFailureInfo;

                    /**
                     * Creates a ChildWorkflowExecutionFailureInfo message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns ChildWorkflowExecutionFailureInfo
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.api.failure.v1.ChildWorkflowExecutionFailureInfo;

                    /**
                     * Creates a plain object from a ChildWorkflowExecutionFailureInfo message. Also converts values to other types if specified.
                     * @param message ChildWorkflowExecutionFailureInfo
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.api.failure.v1.ChildWorkflowExecutionFailureInfo, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this ChildWorkflowExecutionFailureInfo to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for ChildWorkflowExecutionFailureInfo
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of a NexusOperationFailureInfo. */
                interface INexusOperationFailureInfo {

                    /** The NexusOperationScheduled event ID. */
                    scheduledEventId?: (Long|null);

                    /** Endpoint name. */
                    endpoint?: (string|null);

                    /** Service name. */
                    service?: (string|null);

                    /** Operation name. */
                    operation?: (string|null);

                    /**
                     * Operation ID - may be empty if the operation completed synchronously.
                     *
                     * Deprecated. Renamed to operation_token.
                     */
                    operationId?: (string|null);

                    /** Operation token - may be empty if the operation completed synchronously. */
                    operationToken?: (string|null);
                }

                /** Represents a NexusOperationFailureInfo. */
                class NexusOperationFailureInfo implements INexusOperationFailureInfo {

                    /**
                     * Constructs a new NexusOperationFailureInfo.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.api.failure.v1.INexusOperationFailureInfo);

                    /** The NexusOperationScheduled event ID. */
                    public scheduledEventId: Long;

                    /** Endpoint name. */
                    public endpoint: string;

                    /** Service name. */
                    public service: string;

                    /** Operation name. */
                    public operation: string;

                    /**
                     * Operation ID - may be empty if the operation completed synchronously.
                     *
                     * Deprecated. Renamed to operation_token.
                     */
                    public operationId: string;

                    /** Operation token - may be empty if the operation completed synchronously. */
                    public operationToken: string;

                    /**
                     * Creates a new NexusOperationFailureInfo instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns NexusOperationFailureInfo instance
                     */
                    public static create(properties?: temporal.api.failure.v1.INexusOperationFailureInfo): temporal.api.failure.v1.NexusOperationFailureInfo;

                    /**
                     * Encodes the specified NexusOperationFailureInfo message. Does not implicitly {@link temporal.api.failure.v1.NexusOperationFailureInfo.verify|verify} messages.
                     * @param message NexusOperationFailureInfo message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.api.failure.v1.INexusOperationFailureInfo, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified NexusOperationFailureInfo message, length delimited. Does not implicitly {@link temporal.api.failure.v1.NexusOperationFailureInfo.verify|verify} messages.
                     * @param message NexusOperationFailureInfo message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.api.failure.v1.INexusOperationFailureInfo, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a NexusOperationFailureInfo message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns NexusOperationFailureInfo
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.failure.v1.NexusOperationFailureInfo;

                    /**
                     * Decodes a NexusOperationFailureInfo message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns NexusOperationFailureInfo
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.failure.v1.NexusOperationFailureInfo;

                    /**
                     * Creates a NexusOperationFailureInfo message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns NexusOperationFailureInfo
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.api.failure.v1.NexusOperationFailureInfo;

                    /**
                     * Creates a plain object from a NexusOperationFailureInfo message. Also converts values to other types if specified.
                     * @param message NexusOperationFailureInfo
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.api.failure.v1.NexusOperationFailureInfo, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this NexusOperationFailureInfo to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for NexusOperationFailureInfo
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of a NexusHandlerFailureInfo. */
                interface INexusHandlerFailureInfo {

                    /**
                     * The Nexus error type as defined in the spec:
                     * https://github.com/nexus-rpc/api/blob/main/SPEC.md#predefined-handler-errors.
                     */
                    type?: (string|null);

                    /** Retry behavior, defaults to the retry behavior of the error type as defined in the spec. */
                    retryBehavior?: (temporal.api.enums.v1.NexusHandlerErrorRetryBehavior|null);
                }

                /** Represents a NexusHandlerFailureInfo. */
                class NexusHandlerFailureInfo implements INexusHandlerFailureInfo {

                    /**
                     * Constructs a new NexusHandlerFailureInfo.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.api.failure.v1.INexusHandlerFailureInfo);

                    /**
                     * The Nexus error type as defined in the spec:
                     * https://github.com/nexus-rpc/api/blob/main/SPEC.md#predefined-handler-errors.
                     */
                    public type: string;

                    /** Retry behavior, defaults to the retry behavior of the error type as defined in the spec. */
                    public retryBehavior: temporal.api.enums.v1.NexusHandlerErrorRetryBehavior;

                    /**
                     * Creates a new NexusHandlerFailureInfo instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns NexusHandlerFailureInfo instance
                     */
                    public static create(properties?: temporal.api.failure.v1.INexusHandlerFailureInfo): temporal.api.failure.v1.NexusHandlerFailureInfo;

                    /**
                     * Encodes the specified NexusHandlerFailureInfo message. Does not implicitly {@link temporal.api.failure.v1.NexusHandlerFailureInfo.verify|verify} messages.
                     * @param message NexusHandlerFailureInfo message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.api.failure.v1.INexusHandlerFailureInfo, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified NexusHandlerFailureInfo message, length delimited. Does not implicitly {@link temporal.api.failure.v1.NexusHandlerFailureInfo.verify|verify} messages.
                     * @param message NexusHandlerFailureInfo message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.api.failure.v1.INexusHandlerFailureInfo, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a NexusHandlerFailureInfo message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns NexusHandlerFailureInfo
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.failure.v1.NexusHandlerFailureInfo;

                    /**
                     * Decodes a NexusHandlerFailureInfo message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns NexusHandlerFailureInfo
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.failure.v1.NexusHandlerFailureInfo;

                    /**
                     * Creates a NexusHandlerFailureInfo message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns NexusHandlerFailureInfo
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.api.failure.v1.NexusHandlerFailureInfo;

                    /**
                     * Creates a plain object from a NexusHandlerFailureInfo message. Also converts values to other types if specified.
                     * @param message NexusHandlerFailureInfo
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.api.failure.v1.NexusHandlerFailureInfo, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this NexusHandlerFailureInfo to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for NexusHandlerFailureInfo
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of a Failure. */
                interface IFailure {

                    /** Failure message */
                    message?: (string|null);

                    /**
                     * The source this Failure originated in, e.g. TypeScriptSDK / JavaSDK
                     * In some SDKs this is used to rehydrate the stack trace into an exception object.
                     */
                    source?: (string|null);

                    /** Failure stackTrace */
                    stackTrace?: (string|null);

                    /**
                     * Alternative way to supply `message` and `stack_trace` and possibly other attributes, used for encryption of
                     * errors originating in user code which might contain sensitive information.
                     * The `encoded_attributes` Payload could represent any serializable object, e.g. JSON object or a `Failure` proto
                     * message.
                     *
                     * SDK authors:
                     * - The SDK should provide a default `encodeFailureAttributes` and `decodeFailureAttributes` implementation that:
                     * - Uses a JSON object to represent `{ message, stack_trace }`.
                     * - Overwrites the original message with "Encoded failure" to indicate that more information could be extracted.
                     * - Overwrites the original stack_trace with an empty string.
                     * - The resulting JSON object is converted to Payload using the default PayloadConverter and should be processed
                     * by the user-provided PayloadCodec
                     *
                     * - If there's demand, we could allow overriding the default SDK implementation to encode other opaque Failure attributes.
                     * (-- api-linter: core::0203::optional=disabled --)
                     */
                    encodedAttributes?: (temporal.api.common.v1.IPayload|null);

                    /** Failure cause */
                    cause?: (temporal.api.failure.v1.IFailure|null);

                    /** Failure applicationFailureInfo */
                    applicationFailureInfo?: (temporal.api.failure.v1.IApplicationFailureInfo|null);

                    /** Failure timeoutFailureInfo */
                    timeoutFailureInfo?: (temporal.api.failure.v1.ITimeoutFailureInfo|null);

                    /** Failure canceledFailureInfo */
                    canceledFailureInfo?: (temporal.api.failure.v1.ICanceledFailureInfo|null);

                    /** Failure terminatedFailureInfo */
                    terminatedFailureInfo?: (temporal.api.failure.v1.ITerminatedFailureInfo|null);

                    /** Failure serverFailureInfo */
                    serverFailureInfo?: (temporal.api.failure.v1.IServerFailureInfo|null);

                    /** Failure resetWorkflowFailureInfo */
                    resetWorkflowFailureInfo?: (temporal.api.failure.v1.IResetWorkflowFailureInfo|null);

                    /** Failure activityFailureInfo */
                    activityFailureInfo?: (temporal.api.failure.v1.IActivityFailureInfo|null);

                    /** Failure childWorkflowExecutionFailureInfo */
                    childWorkflowExecutionFailureInfo?: (temporal.api.failure.v1.IChildWorkflowExecutionFailureInfo|null);

                    /** Failure nexusOperationExecutionFailureInfo */
                    nexusOperationExecutionFailureInfo?: (temporal.api.failure.v1.INexusOperationFailureInfo|null);

                    /** Failure nexusHandlerFailureInfo */
                    nexusHandlerFailureInfo?: (temporal.api.failure.v1.INexusHandlerFailureInfo|null);
                }

                /** Represents a Failure. */
                class Failure implements IFailure {

                    /**
                     * Constructs a new Failure.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.api.failure.v1.IFailure);

                    /** Failure message. */
                    public message: string;

                    /**
                     * The source this Failure originated in, e.g. TypeScriptSDK / JavaSDK
                     * In some SDKs this is used to rehydrate the stack trace into an exception object.
                     */
                    public source: string;

                    /** Failure stackTrace. */
                    public stackTrace: string;

                    /**
                     * Alternative way to supply `message` and `stack_trace` and possibly other attributes, used for encryption of
                     * errors originating in user code which might contain sensitive information.
                     * The `encoded_attributes` Payload could represent any serializable object, e.g. JSON object or a `Failure` proto
                     * message.
                     *
                     * SDK authors:
                     * - The SDK should provide a default `encodeFailureAttributes` and `decodeFailureAttributes` implementation that:
                     * - Uses a JSON object to represent `{ message, stack_trace }`.
                     * - Overwrites the original message with "Encoded failure" to indicate that more information could be extracted.
                     * - Overwrites the original stack_trace with an empty string.
                     * - The resulting JSON object is converted to Payload using the default PayloadConverter and should be processed
                     * by the user-provided PayloadCodec
                     *
                     * - If there's demand, we could allow overriding the default SDK implementation to encode other opaque Failure attributes.
                     * (-- api-linter: core::0203::optional=disabled --)
                     */
                    public encodedAttributes?: (temporal.api.common.v1.IPayload|null);

                    /** Failure cause. */
                    public cause?: (temporal.api.failure.v1.IFailure|null);

                    /** Failure applicationFailureInfo. */
                    public applicationFailureInfo?: (temporal.api.failure.v1.IApplicationFailureInfo|null);

                    /** Failure timeoutFailureInfo. */
                    public timeoutFailureInfo?: (temporal.api.failure.v1.ITimeoutFailureInfo|null);

                    /** Failure canceledFailureInfo. */
                    public canceledFailureInfo?: (temporal.api.failure.v1.ICanceledFailureInfo|null);

                    /** Failure terminatedFailureInfo. */
                    public terminatedFailureInfo?: (temporal.api.failure.v1.ITerminatedFailureInfo|null);

                    /** Failure serverFailureInfo. */
                    public serverFailureInfo?: (temporal.api.failure.v1.IServerFailureInfo|null);

                    /** Failure resetWorkflowFailureInfo. */
                    public resetWorkflowFailureInfo?: (temporal.api.failure.v1.IResetWorkflowFailureInfo|null);

                    /** Failure activityFailureInfo. */
                    public activityFailureInfo?: (temporal.api.failure.v1.IActivityFailureInfo|null);

                    /** Failure childWorkflowExecutionFailureInfo. */
                    public childWorkflowExecutionFailureInfo?: (temporal.api.failure.v1.IChildWorkflowExecutionFailureInfo|null);

                    /** Failure nexusOperationExecutionFailureInfo. */
                    public nexusOperationExecutionFailureInfo?: (temporal.api.failure.v1.INexusOperationFailureInfo|null);

                    /** Failure nexusHandlerFailureInfo. */
                    public nexusHandlerFailureInfo?: (temporal.api.failure.v1.INexusHandlerFailureInfo|null);

                    /** Failure failureInfo. */
                    public failureInfo?: ("applicationFailureInfo"|"timeoutFailureInfo"|"canceledFailureInfo"|"terminatedFailureInfo"|"serverFailureInfo"|"resetWorkflowFailureInfo"|"activityFailureInfo"|"childWorkflowExecutionFailureInfo"|"nexusOperationExecutionFailureInfo"|"nexusHandlerFailureInfo");

                    /**
                     * Creates a new Failure instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns Failure instance
                     */
                    public static create(properties?: temporal.api.failure.v1.IFailure): temporal.api.failure.v1.Failure;

                    /**
                     * Encodes the specified Failure message. Does not implicitly {@link temporal.api.failure.v1.Failure.verify|verify} messages.
                     * @param message Failure message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.api.failure.v1.IFailure, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified Failure message, length delimited. Does not implicitly {@link temporal.api.failure.v1.Failure.verify|verify} messages.
                     * @param message Failure message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.api.failure.v1.IFailure, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a Failure message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns Failure
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.failure.v1.Failure;

                    /**
                     * Decodes a Failure message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns Failure
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.failure.v1.Failure;

                    /**
                     * Creates a Failure message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns Failure
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.api.failure.v1.Failure;

                    /**
                     * Creates a plain object from a Failure message. Also converts values to other types if specified.
                     * @param message Failure
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.api.failure.v1.Failure, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this Failure to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for Failure
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }

                /** Properties of a MultiOperationExecutionAborted. */
                interface IMultiOperationExecutionAborted {
                }

                /** Represents a MultiOperationExecutionAborted. */
                class MultiOperationExecutionAborted implements IMultiOperationExecutionAborted {

                    /**
                     * Constructs a new MultiOperationExecutionAborted.
                     * @param [properties] Properties to set
                     */
                    constructor(properties?: temporal.api.failure.v1.IMultiOperationExecutionAborted);

                    /**
                     * Creates a new MultiOperationExecutionAborted instance using the specified properties.
                     * @param [properties] Properties to set
                     * @returns MultiOperationExecutionAborted instance
                     */
                    public static create(properties?: temporal.api.failure.v1.IMultiOperationExecutionAborted): temporal.api.failure.v1.MultiOperationExecutionAborted;

                    /**
                     * Encodes the specified MultiOperationExecutionAborted message. Does not implicitly {@link temporal.api.failure.v1.MultiOperationExecutionAborted.verify|verify} messages.
                     * @param message MultiOperationExecutionAborted message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encode(message: temporal.api.failure.v1.IMultiOperationExecutionAborted, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Encodes the specified MultiOperationExecutionAborted message, length delimited. Does not implicitly {@link temporal.api.failure.v1.MultiOperationExecutionAborted.verify|verify} messages.
                     * @param message MultiOperationExecutionAborted message or plain object to encode
                     * @param [writer] Writer to encode to
                     * @returns Writer
                     */
                    public static encodeDelimited(message: temporal.api.failure.v1.IMultiOperationExecutionAborted, writer?: $protobuf.Writer): $protobuf.Writer;

                    /**
                     * Decodes a MultiOperationExecutionAborted message from the specified reader or buffer.
                     * @param reader Reader or buffer to decode from
                     * @param [length] Message length if known beforehand
                     * @returns MultiOperationExecutionAborted
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): temporal.api.failure.v1.MultiOperationExecutionAborted;

                    /**
                     * Decodes a MultiOperationExecutionAborted message from the specified reader or buffer, length delimited.
                     * @param reader Reader or buffer to decode from
                     * @returns MultiOperationExecutionAborted
                     * @throws {Error} If the payload is not a reader or valid buffer
                     * @throws {$protobuf.util.ProtocolError} If required fields are missing
                     */
                    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): temporal.api.failure.v1.MultiOperationExecutionAborted;

                    /**
                     * Creates a MultiOperationExecutionAborted message from a plain object. Also converts values to their respective internal types.
                     * @param object Plain object
                     * @returns MultiOperationExecutionAborted
                     */
                    public static fromObject(object: { [k: string]: any }): temporal.api.failure.v1.MultiOperationExecutionAborted;

                    /**
                     * Creates a plain object from a MultiOperationExecutionAborted message. Also converts values to other types if specified.
                     * @param message MultiOperationExecutionAborted
                     * @param [options] Conversion options
                     * @returns Plain object
                     */
                    public static toObject(message: temporal.api.failure.v1.MultiOperationExecutionAborted, options?: $protobuf.IConversionOptions): { [k: string]: any };

                    /**
                     * Converts this MultiOperationExecutionAborted to JSON.
                     * @returns JSON object
                     */
                    public toJSON(): { [k: string]: any };

                    /**
                     * Gets the default type url for MultiOperationExecutionAborted
                     * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                     * @returns The default type url
                     */
                    public static getTypeUrl(typeUrlPrefix?: string): string;
                }
            }
        }
    }
}

/** Namespace google. */
export namespace google {

    /** Namespace protobuf. */
    namespace protobuf {

        /** Properties of a FileDescriptorSet. */
        interface IFileDescriptorSet {

            /** FileDescriptorSet file */
            file?: (google.protobuf.IFileDescriptorProto[]|null);
        }

        /** Represents a FileDescriptorSet. */
        class FileDescriptorSet implements IFileDescriptorSet {

            /**
             * Constructs a new FileDescriptorSet.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IFileDescriptorSet);

            /** FileDescriptorSet file. */
            public file: google.protobuf.IFileDescriptorProto[];

            /**
             * Creates a new FileDescriptorSet instance using the specified properties.
             * @param [properties] Properties to set
             * @returns FileDescriptorSet instance
             */
            public static create(properties?: google.protobuf.IFileDescriptorSet): google.protobuf.FileDescriptorSet;

            /**
             * Encodes the specified FileDescriptorSet message. Does not implicitly {@link google.protobuf.FileDescriptorSet.verify|verify} messages.
             * @param message FileDescriptorSet message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IFileDescriptorSet, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified FileDescriptorSet message, length delimited. Does not implicitly {@link google.protobuf.FileDescriptorSet.verify|verify} messages.
             * @param message FileDescriptorSet message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IFileDescriptorSet, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a FileDescriptorSet message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns FileDescriptorSet
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.FileDescriptorSet;

            /**
             * Decodes a FileDescriptorSet message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns FileDescriptorSet
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.FileDescriptorSet;

            /**
             * Creates a FileDescriptorSet message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns FileDescriptorSet
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.FileDescriptorSet;

            /**
             * Creates a plain object from a FileDescriptorSet message. Also converts values to other types if specified.
             * @param message FileDescriptorSet
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.FileDescriptorSet, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this FileDescriptorSet to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for FileDescriptorSet
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }

        /** Properties of a FileDescriptorProto. */
        interface IFileDescriptorProto {

            /** FileDescriptorProto name */
            name?: (string|null);

            /** FileDescriptorProto package */
            "package"?: (string|null);

            /** FileDescriptorProto dependency */
            dependency?: (string[]|null);

            /** FileDescriptorProto publicDependency */
            publicDependency?: (number[]|null);

            /** FileDescriptorProto weakDependency */
            weakDependency?: (number[]|null);

            /** FileDescriptorProto messageType */
            messageType?: (google.protobuf.IDescriptorProto[]|null);

            /** FileDescriptorProto enumType */
            enumType?: (google.protobuf.IEnumDescriptorProto[]|null);

            /** FileDescriptorProto service */
            service?: (google.protobuf.IServiceDescriptorProto[]|null);

            /** FileDescriptorProto extension */
            extension?: (google.protobuf.IFieldDescriptorProto[]|null);

            /** FileDescriptorProto options */
            options?: (google.protobuf.IFileOptions|null);

            /** FileDescriptorProto sourceCodeInfo */
            sourceCodeInfo?: (google.protobuf.ISourceCodeInfo|null);

            /** FileDescriptorProto syntax */
            syntax?: (string|null);
        }

        /** Represents a FileDescriptorProto. */
        class FileDescriptorProto implements IFileDescriptorProto {

            /**
             * Constructs a new FileDescriptorProto.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IFileDescriptorProto);

            /** FileDescriptorProto name. */
            public name: string;

            /** FileDescriptorProto package. */
            public package: string;

            /** FileDescriptorProto dependency. */
            public dependency: string[];

            /** FileDescriptorProto publicDependency. */
            public publicDependency: number[];

            /** FileDescriptorProto weakDependency. */
            public weakDependency: number[];

            /** FileDescriptorProto messageType. */
            public messageType: google.protobuf.IDescriptorProto[];

            /** FileDescriptorProto enumType. */
            public enumType: google.protobuf.IEnumDescriptorProto[];

            /** FileDescriptorProto service. */
            public service: google.protobuf.IServiceDescriptorProto[];

            /** FileDescriptorProto extension. */
            public extension: google.protobuf.IFieldDescriptorProto[];

            /** FileDescriptorProto options. */
            public options?: (google.protobuf.IFileOptions|null);

            /** FileDescriptorProto sourceCodeInfo. */
            public sourceCodeInfo?: (google.protobuf.ISourceCodeInfo|null);

            /** FileDescriptorProto syntax. */
            public syntax: string;

            /**
             * Creates a new FileDescriptorProto instance using the specified properties.
             * @param [properties] Properties to set
             * @returns FileDescriptorProto instance
             */
            public static create(properties?: google.protobuf.IFileDescriptorProto): google.protobuf.FileDescriptorProto;

            /**
             * Encodes the specified FileDescriptorProto message. Does not implicitly {@link google.protobuf.FileDescriptorProto.verify|verify} messages.
             * @param message FileDescriptorProto message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IFileDescriptorProto, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified FileDescriptorProto message, length delimited. Does not implicitly {@link google.protobuf.FileDescriptorProto.verify|verify} messages.
             * @param message FileDescriptorProto message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IFileDescriptorProto, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a FileDescriptorProto message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns FileDescriptorProto
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.FileDescriptorProto;

            /**
             * Decodes a FileDescriptorProto message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns FileDescriptorProto
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.FileDescriptorProto;

            /**
             * Creates a FileDescriptorProto message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns FileDescriptorProto
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.FileDescriptorProto;

            /**
             * Creates a plain object from a FileDescriptorProto message. Also converts values to other types if specified.
             * @param message FileDescriptorProto
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.FileDescriptorProto, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this FileDescriptorProto to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for FileDescriptorProto
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }

        /** Properties of a DescriptorProto. */
        interface IDescriptorProto {

            /** DescriptorProto name */
            name?: (string|null);

            /** DescriptorProto field */
            field?: (google.protobuf.IFieldDescriptorProto[]|null);

            /** DescriptorProto extension */
            extension?: (google.protobuf.IFieldDescriptorProto[]|null);

            /** DescriptorProto nestedType */
            nestedType?: (google.protobuf.IDescriptorProto[]|null);

            /** DescriptorProto enumType */
            enumType?: (google.protobuf.IEnumDescriptorProto[]|null);

            /** DescriptorProto extensionRange */
            extensionRange?: (google.protobuf.DescriptorProto.IExtensionRange[]|null);

            /** DescriptorProto oneofDecl */
            oneofDecl?: (google.protobuf.IOneofDescriptorProto[]|null);

            /** DescriptorProto options */
            options?: (google.protobuf.IMessageOptions|null);

            /** DescriptorProto reservedRange */
            reservedRange?: (google.protobuf.DescriptorProto.IReservedRange[]|null);

            /** DescriptorProto reservedName */
            reservedName?: (string[]|null);
        }

        /** Represents a DescriptorProto. */
        class DescriptorProto implements IDescriptorProto {

            /**
             * Constructs a new DescriptorProto.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IDescriptorProto);

            /** DescriptorProto name. */
            public name: string;

            /** DescriptorProto field. */
            public field: google.protobuf.IFieldDescriptorProto[];

            /** DescriptorProto extension. */
            public extension: google.protobuf.IFieldDescriptorProto[];

            /** DescriptorProto nestedType. */
            public nestedType: google.protobuf.IDescriptorProto[];

            /** DescriptorProto enumType. */
            public enumType: google.protobuf.IEnumDescriptorProto[];

            /** DescriptorProto extensionRange. */
            public extensionRange: google.protobuf.DescriptorProto.IExtensionRange[];

            /** DescriptorProto oneofDecl. */
            public oneofDecl: google.protobuf.IOneofDescriptorProto[];

            /** DescriptorProto options. */
            public options?: (google.protobuf.IMessageOptions|null);

            /** DescriptorProto reservedRange. */
            public reservedRange: google.protobuf.DescriptorProto.IReservedRange[];

            /** DescriptorProto reservedName. */
            public reservedName: string[];

            /**
             * Creates a new DescriptorProto instance using the specified properties.
             * @param [properties] Properties to set
             * @returns DescriptorProto instance
             */
            public static create(properties?: google.protobuf.IDescriptorProto): google.protobuf.DescriptorProto;

            /**
             * Encodes the specified DescriptorProto message. Does not implicitly {@link google.protobuf.DescriptorProto.verify|verify} messages.
             * @param message DescriptorProto message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IDescriptorProto, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified DescriptorProto message, length delimited. Does not implicitly {@link google.protobuf.DescriptorProto.verify|verify} messages.
             * @param message DescriptorProto message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IDescriptorProto, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a DescriptorProto message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns DescriptorProto
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.DescriptorProto;

            /**
             * Decodes a DescriptorProto message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns DescriptorProto
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.DescriptorProto;

            /**
             * Creates a DescriptorProto message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns DescriptorProto
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.DescriptorProto;

            /**
             * Creates a plain object from a DescriptorProto message. Also converts values to other types if specified.
             * @param message DescriptorProto
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.DescriptorProto, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this DescriptorProto to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for DescriptorProto
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }

        namespace DescriptorProto {

            /** Properties of an ExtensionRange. */
            interface IExtensionRange {

                /** ExtensionRange start */
                start?: (number|null);

                /** ExtensionRange end */
                end?: (number|null);
            }

            /** Represents an ExtensionRange. */
            class ExtensionRange implements IExtensionRange {

                /**
                 * Constructs a new ExtensionRange.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: google.protobuf.DescriptorProto.IExtensionRange);

                /** ExtensionRange start. */
                public start: number;

                /** ExtensionRange end. */
                public end: number;

                /**
                 * Creates a new ExtensionRange instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns ExtensionRange instance
                 */
                public static create(properties?: google.protobuf.DescriptorProto.IExtensionRange): google.protobuf.DescriptorProto.ExtensionRange;

                /**
                 * Encodes the specified ExtensionRange message. Does not implicitly {@link google.protobuf.DescriptorProto.ExtensionRange.verify|verify} messages.
                 * @param message ExtensionRange message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: google.protobuf.DescriptorProto.IExtensionRange, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified ExtensionRange message, length delimited. Does not implicitly {@link google.protobuf.DescriptorProto.ExtensionRange.verify|verify} messages.
                 * @param message ExtensionRange message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: google.protobuf.DescriptorProto.IExtensionRange, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes an ExtensionRange message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns ExtensionRange
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.DescriptorProto.ExtensionRange;

                /**
                 * Decodes an ExtensionRange message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns ExtensionRange
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.DescriptorProto.ExtensionRange;

                /**
                 * Creates an ExtensionRange message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns ExtensionRange
                 */
                public static fromObject(object: { [k: string]: any }): google.protobuf.DescriptorProto.ExtensionRange;

                /**
                 * Creates a plain object from an ExtensionRange message. Also converts values to other types if specified.
                 * @param message ExtensionRange
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: google.protobuf.DescriptorProto.ExtensionRange, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this ExtensionRange to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for ExtensionRange
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }

            /** Properties of a ReservedRange. */
            interface IReservedRange {

                /** ReservedRange start */
                start?: (number|null);

                /** ReservedRange end */
                end?: (number|null);
            }

            /** Represents a ReservedRange. */
            class ReservedRange implements IReservedRange {

                /**
                 * Constructs a new ReservedRange.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: google.protobuf.DescriptorProto.IReservedRange);

                /** ReservedRange start. */
                public start: number;

                /** ReservedRange end. */
                public end: number;

                /**
                 * Creates a new ReservedRange instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns ReservedRange instance
                 */
                public static create(properties?: google.protobuf.DescriptorProto.IReservedRange): google.protobuf.DescriptorProto.ReservedRange;

                /**
                 * Encodes the specified ReservedRange message. Does not implicitly {@link google.protobuf.DescriptorProto.ReservedRange.verify|verify} messages.
                 * @param message ReservedRange message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: google.protobuf.DescriptorProto.IReservedRange, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified ReservedRange message, length delimited. Does not implicitly {@link google.protobuf.DescriptorProto.ReservedRange.verify|verify} messages.
                 * @param message ReservedRange message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: google.protobuf.DescriptorProto.IReservedRange, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a ReservedRange message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns ReservedRange
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.DescriptorProto.ReservedRange;

                /**
                 * Decodes a ReservedRange message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns ReservedRange
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.DescriptorProto.ReservedRange;

                /**
                 * Creates a ReservedRange message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns ReservedRange
                 */
                public static fromObject(object: { [k: string]: any }): google.protobuf.DescriptorProto.ReservedRange;

                /**
                 * Creates a plain object from a ReservedRange message. Also converts values to other types if specified.
                 * @param message ReservedRange
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: google.protobuf.DescriptorProto.ReservedRange, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this ReservedRange to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for ReservedRange
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }
        }

        /** Properties of a FieldDescriptorProto. */
        interface IFieldDescriptorProto {

            /** FieldDescriptorProto name */
            name?: (string|null);

            /** FieldDescriptorProto number */
            number?: (number|null);

            /** FieldDescriptorProto label */
            label?: (google.protobuf.FieldDescriptorProto.Label|null);

            /** FieldDescriptorProto type */
            type?: (google.protobuf.FieldDescriptorProto.Type|null);

            /** FieldDescriptorProto typeName */
            typeName?: (string|null);

            /** FieldDescriptorProto extendee */
            extendee?: (string|null);

            /** FieldDescriptorProto defaultValue */
            defaultValue?: (string|null);

            /** FieldDescriptorProto oneofIndex */
            oneofIndex?: (number|null);

            /** FieldDescriptorProto jsonName */
            jsonName?: (string|null);

            /** FieldDescriptorProto options */
            options?: (google.protobuf.IFieldOptions|null);
        }

        /** Represents a FieldDescriptorProto. */
        class FieldDescriptorProto implements IFieldDescriptorProto {

            /**
             * Constructs a new FieldDescriptorProto.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IFieldDescriptorProto);

            /** FieldDescriptorProto name. */
            public name: string;

            /** FieldDescriptorProto number. */
            public number: number;

            /** FieldDescriptorProto label. */
            public label: google.protobuf.FieldDescriptorProto.Label;

            /** FieldDescriptorProto type. */
            public type: google.protobuf.FieldDescriptorProto.Type;

            /** FieldDescriptorProto typeName. */
            public typeName: string;

            /** FieldDescriptorProto extendee. */
            public extendee: string;

            /** FieldDescriptorProto defaultValue. */
            public defaultValue: string;

            /** FieldDescriptorProto oneofIndex. */
            public oneofIndex: number;

            /** FieldDescriptorProto jsonName. */
            public jsonName: string;

            /** FieldDescriptorProto options. */
            public options?: (google.protobuf.IFieldOptions|null);

            /**
             * Creates a new FieldDescriptorProto instance using the specified properties.
             * @param [properties] Properties to set
             * @returns FieldDescriptorProto instance
             */
            public static create(properties?: google.protobuf.IFieldDescriptorProto): google.protobuf.FieldDescriptorProto;

            /**
             * Encodes the specified FieldDescriptorProto message. Does not implicitly {@link google.protobuf.FieldDescriptorProto.verify|verify} messages.
             * @param message FieldDescriptorProto message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IFieldDescriptorProto, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified FieldDescriptorProto message, length delimited. Does not implicitly {@link google.protobuf.FieldDescriptorProto.verify|verify} messages.
             * @param message FieldDescriptorProto message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IFieldDescriptorProto, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a FieldDescriptorProto message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns FieldDescriptorProto
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.FieldDescriptorProto;

            /**
             * Decodes a FieldDescriptorProto message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns FieldDescriptorProto
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.FieldDescriptorProto;

            /**
             * Creates a FieldDescriptorProto message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns FieldDescriptorProto
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.FieldDescriptorProto;

            /**
             * Creates a plain object from a FieldDescriptorProto message. Also converts values to other types if specified.
             * @param message FieldDescriptorProto
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.FieldDescriptorProto, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this FieldDescriptorProto to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for FieldDescriptorProto
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }

        namespace FieldDescriptorProto {

            /** Type enum. */
            enum Type {
                TYPE_DOUBLE = 1,
                TYPE_FLOAT = 2,
                TYPE_INT64 = 3,
                TYPE_UINT64 = 4,
                TYPE_INT32 = 5,
                TYPE_FIXED64 = 6,
                TYPE_FIXED32 = 7,
                TYPE_BOOL = 8,
                TYPE_STRING = 9,
                TYPE_GROUP = 10,
                TYPE_MESSAGE = 11,
                TYPE_BYTES = 12,
                TYPE_UINT32 = 13,
                TYPE_ENUM = 14,
                TYPE_SFIXED32 = 15,
                TYPE_SFIXED64 = 16,
                TYPE_SINT32 = 17,
                TYPE_SINT64 = 18
            }

            /** Label enum. */
            enum Label {
                LABEL_OPTIONAL = 1,
                LABEL_REQUIRED = 2,
                LABEL_REPEATED = 3
            }
        }

        /** Properties of an OneofDescriptorProto. */
        interface IOneofDescriptorProto {

            /** OneofDescriptorProto name */
            name?: (string|null);

            /** OneofDescriptorProto options */
            options?: (google.protobuf.IOneofOptions|null);
        }

        /** Represents an OneofDescriptorProto. */
        class OneofDescriptorProto implements IOneofDescriptorProto {

            /**
             * Constructs a new OneofDescriptorProto.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IOneofDescriptorProto);

            /** OneofDescriptorProto name. */
            public name: string;

            /** OneofDescriptorProto options. */
            public options?: (google.protobuf.IOneofOptions|null);

            /**
             * Creates a new OneofDescriptorProto instance using the specified properties.
             * @param [properties] Properties to set
             * @returns OneofDescriptorProto instance
             */
            public static create(properties?: google.protobuf.IOneofDescriptorProto): google.protobuf.OneofDescriptorProto;

            /**
             * Encodes the specified OneofDescriptorProto message. Does not implicitly {@link google.protobuf.OneofDescriptorProto.verify|verify} messages.
             * @param message OneofDescriptorProto message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IOneofDescriptorProto, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified OneofDescriptorProto message, length delimited. Does not implicitly {@link google.protobuf.OneofDescriptorProto.verify|verify} messages.
             * @param message OneofDescriptorProto message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IOneofDescriptorProto, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes an OneofDescriptorProto message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns OneofDescriptorProto
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.OneofDescriptorProto;

            /**
             * Decodes an OneofDescriptorProto message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns OneofDescriptorProto
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.OneofDescriptorProto;

            /**
             * Creates an OneofDescriptorProto message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns OneofDescriptorProto
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.OneofDescriptorProto;

            /**
             * Creates a plain object from an OneofDescriptorProto message. Also converts values to other types if specified.
             * @param message OneofDescriptorProto
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.OneofDescriptorProto, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this OneofDescriptorProto to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for OneofDescriptorProto
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }

        /** Properties of an EnumDescriptorProto. */
        interface IEnumDescriptorProto {

            /** EnumDescriptorProto name */
            name?: (string|null);

            /** EnumDescriptorProto value */
            value?: (google.protobuf.IEnumValueDescriptorProto[]|null);

            /** EnumDescriptorProto options */
            options?: (google.protobuf.IEnumOptions|null);
        }

        /** Represents an EnumDescriptorProto. */
        class EnumDescriptorProto implements IEnumDescriptorProto {

            /**
             * Constructs a new EnumDescriptorProto.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IEnumDescriptorProto);

            /** EnumDescriptorProto name. */
            public name: string;

            /** EnumDescriptorProto value. */
            public value: google.protobuf.IEnumValueDescriptorProto[];

            /** EnumDescriptorProto options. */
            public options?: (google.protobuf.IEnumOptions|null);

            /**
             * Creates a new EnumDescriptorProto instance using the specified properties.
             * @param [properties] Properties to set
             * @returns EnumDescriptorProto instance
             */
            public static create(properties?: google.protobuf.IEnumDescriptorProto): google.protobuf.EnumDescriptorProto;

            /**
             * Encodes the specified EnumDescriptorProto message. Does not implicitly {@link google.protobuf.EnumDescriptorProto.verify|verify} messages.
             * @param message EnumDescriptorProto message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IEnumDescriptorProto, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified EnumDescriptorProto message, length delimited. Does not implicitly {@link google.protobuf.EnumDescriptorProto.verify|verify} messages.
             * @param message EnumDescriptorProto message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IEnumDescriptorProto, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes an EnumDescriptorProto message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns EnumDescriptorProto
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.EnumDescriptorProto;

            /**
             * Decodes an EnumDescriptorProto message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns EnumDescriptorProto
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.EnumDescriptorProto;

            /**
             * Creates an EnumDescriptorProto message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns EnumDescriptorProto
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.EnumDescriptorProto;

            /**
             * Creates a plain object from an EnumDescriptorProto message. Also converts values to other types if specified.
             * @param message EnumDescriptorProto
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.EnumDescriptorProto, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this EnumDescriptorProto to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for EnumDescriptorProto
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }

        /** Properties of an EnumValueDescriptorProto. */
        interface IEnumValueDescriptorProto {

            /** EnumValueDescriptorProto name */
            name?: (string|null);

            /** EnumValueDescriptorProto number */
            number?: (number|null);

            /** EnumValueDescriptorProto options */
            options?: (google.protobuf.IEnumValueOptions|null);
        }

        /** Represents an EnumValueDescriptorProto. */
        class EnumValueDescriptorProto implements IEnumValueDescriptorProto {

            /**
             * Constructs a new EnumValueDescriptorProto.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IEnumValueDescriptorProto);

            /** EnumValueDescriptorProto name. */
            public name: string;

            /** EnumValueDescriptorProto number. */
            public number: number;

            /** EnumValueDescriptorProto options. */
            public options?: (google.protobuf.IEnumValueOptions|null);

            /**
             * Creates a new EnumValueDescriptorProto instance using the specified properties.
             * @param [properties] Properties to set
             * @returns EnumValueDescriptorProto instance
             */
            public static create(properties?: google.protobuf.IEnumValueDescriptorProto): google.protobuf.EnumValueDescriptorProto;

            /**
             * Encodes the specified EnumValueDescriptorProto message. Does not implicitly {@link google.protobuf.EnumValueDescriptorProto.verify|verify} messages.
             * @param message EnumValueDescriptorProto message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IEnumValueDescriptorProto, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified EnumValueDescriptorProto message, length delimited. Does not implicitly {@link google.protobuf.EnumValueDescriptorProto.verify|verify} messages.
             * @param message EnumValueDescriptorProto message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IEnumValueDescriptorProto, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes an EnumValueDescriptorProto message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns EnumValueDescriptorProto
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.EnumValueDescriptorProto;

            /**
             * Decodes an EnumValueDescriptorProto message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns EnumValueDescriptorProto
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.EnumValueDescriptorProto;

            /**
             * Creates an EnumValueDescriptorProto message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns EnumValueDescriptorProto
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.EnumValueDescriptorProto;

            /**
             * Creates a plain object from an EnumValueDescriptorProto message. Also converts values to other types if specified.
             * @param message EnumValueDescriptorProto
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.EnumValueDescriptorProto, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this EnumValueDescriptorProto to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for EnumValueDescriptorProto
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }

        /** Properties of a ServiceDescriptorProto. */
        interface IServiceDescriptorProto {

            /** ServiceDescriptorProto name */
            name?: (string|null);

            /** ServiceDescriptorProto method */
            method?: (google.protobuf.IMethodDescriptorProto[]|null);

            /** ServiceDescriptorProto options */
            options?: (google.protobuf.IServiceOptions|null);
        }

        /** Represents a ServiceDescriptorProto. */
        class ServiceDescriptorProto implements IServiceDescriptorProto {

            /**
             * Constructs a new ServiceDescriptorProto.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IServiceDescriptorProto);

            /** ServiceDescriptorProto name. */
            public name: string;

            /** ServiceDescriptorProto method. */
            public method: google.protobuf.IMethodDescriptorProto[];

            /** ServiceDescriptorProto options. */
            public options?: (google.protobuf.IServiceOptions|null);

            /**
             * Creates a new ServiceDescriptorProto instance using the specified properties.
             * @param [properties] Properties to set
             * @returns ServiceDescriptorProto instance
             */
            public static create(properties?: google.protobuf.IServiceDescriptorProto): google.protobuf.ServiceDescriptorProto;

            /**
             * Encodes the specified ServiceDescriptorProto message. Does not implicitly {@link google.protobuf.ServiceDescriptorProto.verify|verify} messages.
             * @param message ServiceDescriptorProto message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IServiceDescriptorProto, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified ServiceDescriptorProto message, length delimited. Does not implicitly {@link google.protobuf.ServiceDescriptorProto.verify|verify} messages.
             * @param message ServiceDescriptorProto message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IServiceDescriptorProto, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a ServiceDescriptorProto message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns ServiceDescriptorProto
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.ServiceDescriptorProto;

            /**
             * Decodes a ServiceDescriptorProto message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns ServiceDescriptorProto
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.ServiceDescriptorProto;

            /**
             * Creates a ServiceDescriptorProto message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns ServiceDescriptorProto
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.ServiceDescriptorProto;

            /**
             * Creates a plain object from a ServiceDescriptorProto message. Also converts values to other types if specified.
             * @param message ServiceDescriptorProto
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.ServiceDescriptorProto, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this ServiceDescriptorProto to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for ServiceDescriptorProto
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }

        /** Properties of a MethodDescriptorProto. */
        interface IMethodDescriptorProto {

            /** MethodDescriptorProto name */
            name?: (string|null);

            /** MethodDescriptorProto inputType */
            inputType?: (string|null);

            /** MethodDescriptorProto outputType */
            outputType?: (string|null);

            /** MethodDescriptorProto options */
            options?: (google.protobuf.IMethodOptions|null);

            /** MethodDescriptorProto clientStreaming */
            clientStreaming?: (boolean|null);

            /** MethodDescriptorProto serverStreaming */
            serverStreaming?: (boolean|null);
        }

        /** Represents a MethodDescriptorProto. */
        class MethodDescriptorProto implements IMethodDescriptorProto {

            /**
             * Constructs a new MethodDescriptorProto.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IMethodDescriptorProto);

            /** MethodDescriptorProto name. */
            public name: string;

            /** MethodDescriptorProto inputType. */
            public inputType: string;

            /** MethodDescriptorProto outputType. */
            public outputType: string;

            /** MethodDescriptorProto options. */
            public options?: (google.protobuf.IMethodOptions|null);

            /** MethodDescriptorProto clientStreaming. */
            public clientStreaming: boolean;

            /** MethodDescriptorProto serverStreaming. */
            public serverStreaming: boolean;

            /**
             * Creates a new MethodDescriptorProto instance using the specified properties.
             * @param [properties] Properties to set
             * @returns MethodDescriptorProto instance
             */
            public static create(properties?: google.protobuf.IMethodDescriptorProto): google.protobuf.MethodDescriptorProto;

            /**
             * Encodes the specified MethodDescriptorProto message. Does not implicitly {@link google.protobuf.MethodDescriptorProto.verify|verify} messages.
             * @param message MethodDescriptorProto message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IMethodDescriptorProto, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified MethodDescriptorProto message, length delimited. Does not implicitly {@link google.protobuf.MethodDescriptorProto.verify|verify} messages.
             * @param message MethodDescriptorProto message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IMethodDescriptorProto, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a MethodDescriptorProto message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns MethodDescriptorProto
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.MethodDescriptorProto;

            /**
             * Decodes a MethodDescriptorProto message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns MethodDescriptorProto
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.MethodDescriptorProto;

            /**
             * Creates a MethodDescriptorProto message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns MethodDescriptorProto
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.MethodDescriptorProto;

            /**
             * Creates a plain object from a MethodDescriptorProto message. Also converts values to other types if specified.
             * @param message MethodDescriptorProto
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.MethodDescriptorProto, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this MethodDescriptorProto to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for MethodDescriptorProto
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }

        /** Properties of a FileOptions. */
        interface IFileOptions {

            /** FileOptions javaPackage */
            javaPackage?: (string|null);

            /** FileOptions javaOuterClassname */
            javaOuterClassname?: (string|null);

            /** FileOptions javaMultipleFiles */
            javaMultipleFiles?: (boolean|null);

            /** FileOptions javaGenerateEqualsAndHash */
            javaGenerateEqualsAndHash?: (boolean|null);

            /** FileOptions javaStringCheckUtf8 */
            javaStringCheckUtf8?: (boolean|null);

            /** FileOptions optimizeFor */
            optimizeFor?: (google.protobuf.FileOptions.OptimizeMode|null);

            /** FileOptions goPackage */
            goPackage?: (string|null);

            /** FileOptions ccGenericServices */
            ccGenericServices?: (boolean|null);

            /** FileOptions javaGenericServices */
            javaGenericServices?: (boolean|null);

            /** FileOptions pyGenericServices */
            pyGenericServices?: (boolean|null);

            /** FileOptions deprecated */
            deprecated?: (boolean|null);

            /** FileOptions ccEnableArenas */
            ccEnableArenas?: (boolean|null);

            /** FileOptions objcClassPrefix */
            objcClassPrefix?: (string|null);

            /** FileOptions csharpNamespace */
            csharpNamespace?: (string|null);

            /** FileOptions uninterpretedOption */
            uninterpretedOption?: (google.protobuf.IUninterpretedOption[]|null);
        }

        /** Represents a FileOptions. */
        class FileOptions implements IFileOptions {

            /**
             * Constructs a new FileOptions.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IFileOptions);

            /** FileOptions javaPackage. */
            public javaPackage: string;

            /** FileOptions javaOuterClassname. */
            public javaOuterClassname: string;

            /** FileOptions javaMultipleFiles. */
            public javaMultipleFiles: boolean;

            /** FileOptions javaGenerateEqualsAndHash. */
            public javaGenerateEqualsAndHash: boolean;

            /** FileOptions javaStringCheckUtf8. */
            public javaStringCheckUtf8: boolean;

            /** FileOptions optimizeFor. */
            public optimizeFor: google.protobuf.FileOptions.OptimizeMode;

            /** FileOptions goPackage. */
            public goPackage: string;

            /** FileOptions ccGenericServices. */
            public ccGenericServices: boolean;

            /** FileOptions javaGenericServices. */
            public javaGenericServices: boolean;

            /** FileOptions pyGenericServices. */
            public pyGenericServices: boolean;

            /** FileOptions deprecated. */
            public deprecated: boolean;

            /** FileOptions ccEnableArenas. */
            public ccEnableArenas: boolean;

            /** FileOptions objcClassPrefix. */
            public objcClassPrefix: string;

            /** FileOptions csharpNamespace. */
            public csharpNamespace: string;

            /** FileOptions uninterpretedOption. */
            public uninterpretedOption: google.protobuf.IUninterpretedOption[];

            /**
             * Creates a new FileOptions instance using the specified properties.
             * @param [properties] Properties to set
             * @returns FileOptions instance
             */
            public static create(properties?: google.protobuf.IFileOptions): google.protobuf.FileOptions;

            /**
             * Encodes the specified FileOptions message. Does not implicitly {@link google.protobuf.FileOptions.verify|verify} messages.
             * @param message FileOptions message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IFileOptions, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified FileOptions message, length delimited. Does not implicitly {@link google.protobuf.FileOptions.verify|verify} messages.
             * @param message FileOptions message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IFileOptions, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a FileOptions message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns FileOptions
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.FileOptions;

            /**
             * Decodes a FileOptions message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns FileOptions
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.FileOptions;

            /**
             * Creates a FileOptions message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns FileOptions
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.FileOptions;

            /**
             * Creates a plain object from a FileOptions message. Also converts values to other types if specified.
             * @param message FileOptions
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.FileOptions, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this FileOptions to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for FileOptions
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }

        namespace FileOptions {

            /** OptimizeMode enum. */
            enum OptimizeMode {
                SPEED = 1,
                CODE_SIZE = 2,
                LITE_RUNTIME = 3
            }
        }

        /** Properties of a MessageOptions. */
        interface IMessageOptions {

            /** MessageOptions messageSetWireFormat */
            messageSetWireFormat?: (boolean|null);

            /** MessageOptions noStandardDescriptorAccessor */
            noStandardDescriptorAccessor?: (boolean|null);

            /** MessageOptions deprecated */
            deprecated?: (boolean|null);

            /** MessageOptions mapEntry */
            mapEntry?: (boolean|null);

            /** MessageOptions uninterpretedOption */
            uninterpretedOption?: (google.protobuf.IUninterpretedOption[]|null);
        }

        /** Represents a MessageOptions. */
        class MessageOptions implements IMessageOptions {

            /**
             * Constructs a new MessageOptions.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IMessageOptions);

            /** MessageOptions messageSetWireFormat. */
            public messageSetWireFormat: boolean;

            /** MessageOptions noStandardDescriptorAccessor. */
            public noStandardDescriptorAccessor: boolean;

            /** MessageOptions deprecated. */
            public deprecated: boolean;

            /** MessageOptions mapEntry. */
            public mapEntry: boolean;

            /** MessageOptions uninterpretedOption. */
            public uninterpretedOption: google.protobuf.IUninterpretedOption[];

            /**
             * Creates a new MessageOptions instance using the specified properties.
             * @param [properties] Properties to set
             * @returns MessageOptions instance
             */
            public static create(properties?: google.protobuf.IMessageOptions): google.protobuf.MessageOptions;

            /**
             * Encodes the specified MessageOptions message. Does not implicitly {@link google.protobuf.MessageOptions.verify|verify} messages.
             * @param message MessageOptions message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IMessageOptions, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified MessageOptions message, length delimited. Does not implicitly {@link google.protobuf.MessageOptions.verify|verify} messages.
             * @param message MessageOptions message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IMessageOptions, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a MessageOptions message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns MessageOptions
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.MessageOptions;

            /**
             * Decodes a MessageOptions message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns MessageOptions
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.MessageOptions;

            /**
             * Creates a MessageOptions message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns MessageOptions
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.MessageOptions;

            /**
             * Creates a plain object from a MessageOptions message. Also converts values to other types if specified.
             * @param message MessageOptions
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.MessageOptions, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this MessageOptions to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for MessageOptions
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }

        /** Properties of a FieldOptions. */
        interface IFieldOptions {

            /** FieldOptions ctype */
            ctype?: (google.protobuf.FieldOptions.CType|null);

            /** FieldOptions packed */
            packed?: (boolean|null);

            /** FieldOptions jstype */
            jstype?: (google.protobuf.FieldOptions.JSType|null);

            /** FieldOptions lazy */
            lazy?: (boolean|null);

            /** FieldOptions deprecated */
            deprecated?: (boolean|null);

            /** FieldOptions weak */
            weak?: (boolean|null);

            /** FieldOptions uninterpretedOption */
            uninterpretedOption?: (google.protobuf.IUninterpretedOption[]|null);
        }

        /** Represents a FieldOptions. */
        class FieldOptions implements IFieldOptions {

            /**
             * Constructs a new FieldOptions.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IFieldOptions);

            /** FieldOptions ctype. */
            public ctype: google.protobuf.FieldOptions.CType;

            /** FieldOptions packed. */
            public packed: boolean;

            /** FieldOptions jstype. */
            public jstype: google.protobuf.FieldOptions.JSType;

            /** FieldOptions lazy. */
            public lazy: boolean;

            /** FieldOptions deprecated. */
            public deprecated: boolean;

            /** FieldOptions weak. */
            public weak: boolean;

            /** FieldOptions uninterpretedOption. */
            public uninterpretedOption: google.protobuf.IUninterpretedOption[];

            /**
             * Creates a new FieldOptions instance using the specified properties.
             * @param [properties] Properties to set
             * @returns FieldOptions instance
             */
            public static create(properties?: google.protobuf.IFieldOptions): google.protobuf.FieldOptions;

            /**
             * Encodes the specified FieldOptions message. Does not implicitly {@link google.protobuf.FieldOptions.verify|verify} messages.
             * @param message FieldOptions message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IFieldOptions, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified FieldOptions message, length delimited. Does not implicitly {@link google.protobuf.FieldOptions.verify|verify} messages.
             * @param message FieldOptions message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IFieldOptions, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a FieldOptions message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns FieldOptions
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.FieldOptions;

            /**
             * Decodes a FieldOptions message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns FieldOptions
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.FieldOptions;

            /**
             * Creates a FieldOptions message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns FieldOptions
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.FieldOptions;

            /**
             * Creates a plain object from a FieldOptions message. Also converts values to other types if specified.
             * @param message FieldOptions
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.FieldOptions, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this FieldOptions to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for FieldOptions
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }

        namespace FieldOptions {

            /** CType enum. */
            enum CType {
                STRING = 0,
                CORD = 1,
                STRING_PIECE = 2
            }

            /** JSType enum. */
            enum JSType {
                JS_NORMAL = 0,
                JS_STRING = 1,
                JS_NUMBER = 2
            }
        }

        /** Properties of an OneofOptions. */
        interface IOneofOptions {

            /** OneofOptions uninterpretedOption */
            uninterpretedOption?: (google.protobuf.IUninterpretedOption[]|null);
        }

        /** Represents an OneofOptions. */
        class OneofOptions implements IOneofOptions {

            /**
             * Constructs a new OneofOptions.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IOneofOptions);

            /** OneofOptions uninterpretedOption. */
            public uninterpretedOption: google.protobuf.IUninterpretedOption[];

            /**
             * Creates a new OneofOptions instance using the specified properties.
             * @param [properties] Properties to set
             * @returns OneofOptions instance
             */
            public static create(properties?: google.protobuf.IOneofOptions): google.protobuf.OneofOptions;

            /**
             * Encodes the specified OneofOptions message. Does not implicitly {@link google.protobuf.OneofOptions.verify|verify} messages.
             * @param message OneofOptions message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IOneofOptions, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified OneofOptions message, length delimited. Does not implicitly {@link google.protobuf.OneofOptions.verify|verify} messages.
             * @param message OneofOptions message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IOneofOptions, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes an OneofOptions message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns OneofOptions
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.OneofOptions;

            /**
             * Decodes an OneofOptions message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns OneofOptions
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.OneofOptions;

            /**
             * Creates an OneofOptions message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns OneofOptions
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.OneofOptions;

            /**
             * Creates a plain object from an OneofOptions message. Also converts values to other types if specified.
             * @param message OneofOptions
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.OneofOptions, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this OneofOptions to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for OneofOptions
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }

        /** Properties of an EnumOptions. */
        interface IEnumOptions {

            /** EnumOptions allowAlias */
            allowAlias?: (boolean|null);

            /** EnumOptions deprecated */
            deprecated?: (boolean|null);

            /** EnumOptions uninterpretedOption */
            uninterpretedOption?: (google.protobuf.IUninterpretedOption[]|null);
        }

        /** Represents an EnumOptions. */
        class EnumOptions implements IEnumOptions {

            /**
             * Constructs a new EnumOptions.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IEnumOptions);

            /** EnumOptions allowAlias. */
            public allowAlias: boolean;

            /** EnumOptions deprecated. */
            public deprecated: boolean;

            /** EnumOptions uninterpretedOption. */
            public uninterpretedOption: google.protobuf.IUninterpretedOption[];

            /**
             * Creates a new EnumOptions instance using the specified properties.
             * @param [properties] Properties to set
             * @returns EnumOptions instance
             */
            public static create(properties?: google.protobuf.IEnumOptions): google.protobuf.EnumOptions;

            /**
             * Encodes the specified EnumOptions message. Does not implicitly {@link google.protobuf.EnumOptions.verify|verify} messages.
             * @param message EnumOptions message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IEnumOptions, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified EnumOptions message, length delimited. Does not implicitly {@link google.protobuf.EnumOptions.verify|verify} messages.
             * @param message EnumOptions message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IEnumOptions, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes an EnumOptions message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns EnumOptions
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.EnumOptions;

            /**
             * Decodes an EnumOptions message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns EnumOptions
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.EnumOptions;

            /**
             * Creates an EnumOptions message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns EnumOptions
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.EnumOptions;

            /**
             * Creates a plain object from an EnumOptions message. Also converts values to other types if specified.
             * @param message EnumOptions
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.EnumOptions, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this EnumOptions to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for EnumOptions
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }

        /** Properties of an EnumValueOptions. */
        interface IEnumValueOptions {

            /** EnumValueOptions deprecated */
            deprecated?: (boolean|null);

            /** EnumValueOptions uninterpretedOption */
            uninterpretedOption?: (google.protobuf.IUninterpretedOption[]|null);
        }

        /** Represents an EnumValueOptions. */
        class EnumValueOptions implements IEnumValueOptions {

            /**
             * Constructs a new EnumValueOptions.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IEnumValueOptions);

            /** EnumValueOptions deprecated. */
            public deprecated: boolean;

            /** EnumValueOptions uninterpretedOption. */
            public uninterpretedOption: google.protobuf.IUninterpretedOption[];

            /**
             * Creates a new EnumValueOptions instance using the specified properties.
             * @param [properties] Properties to set
             * @returns EnumValueOptions instance
             */
            public static create(properties?: google.protobuf.IEnumValueOptions): google.protobuf.EnumValueOptions;

            /**
             * Encodes the specified EnumValueOptions message. Does not implicitly {@link google.protobuf.EnumValueOptions.verify|verify} messages.
             * @param message EnumValueOptions message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IEnumValueOptions, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified EnumValueOptions message, length delimited. Does not implicitly {@link google.protobuf.EnumValueOptions.verify|verify} messages.
             * @param message EnumValueOptions message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IEnumValueOptions, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes an EnumValueOptions message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns EnumValueOptions
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.EnumValueOptions;

            /**
             * Decodes an EnumValueOptions message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns EnumValueOptions
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.EnumValueOptions;

            /**
             * Creates an EnumValueOptions message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns EnumValueOptions
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.EnumValueOptions;

            /**
             * Creates a plain object from an EnumValueOptions message. Also converts values to other types if specified.
             * @param message EnumValueOptions
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.EnumValueOptions, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this EnumValueOptions to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for EnumValueOptions
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }

        /** Properties of a ServiceOptions. */
        interface IServiceOptions {

            /** ServiceOptions deprecated */
            deprecated?: (boolean|null);

            /** ServiceOptions uninterpretedOption */
            uninterpretedOption?: (google.protobuf.IUninterpretedOption[]|null);
        }

        /** Represents a ServiceOptions. */
        class ServiceOptions implements IServiceOptions {

            /**
             * Constructs a new ServiceOptions.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IServiceOptions);

            /** ServiceOptions deprecated. */
            public deprecated: boolean;

            /** ServiceOptions uninterpretedOption. */
            public uninterpretedOption: google.protobuf.IUninterpretedOption[];

            /**
             * Creates a new ServiceOptions instance using the specified properties.
             * @param [properties] Properties to set
             * @returns ServiceOptions instance
             */
            public static create(properties?: google.protobuf.IServiceOptions): google.protobuf.ServiceOptions;

            /**
             * Encodes the specified ServiceOptions message. Does not implicitly {@link google.protobuf.ServiceOptions.verify|verify} messages.
             * @param message ServiceOptions message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IServiceOptions, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified ServiceOptions message, length delimited. Does not implicitly {@link google.protobuf.ServiceOptions.verify|verify} messages.
             * @param message ServiceOptions message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IServiceOptions, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a ServiceOptions message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns ServiceOptions
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.ServiceOptions;

            /**
             * Decodes a ServiceOptions message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns ServiceOptions
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.ServiceOptions;

            /**
             * Creates a ServiceOptions message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns ServiceOptions
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.ServiceOptions;

            /**
             * Creates a plain object from a ServiceOptions message. Also converts values to other types if specified.
             * @param message ServiceOptions
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.ServiceOptions, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this ServiceOptions to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for ServiceOptions
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }

        /** Properties of a MethodOptions. */
        interface IMethodOptions {

            /** MethodOptions deprecated */
            deprecated?: (boolean|null);

            /** MethodOptions uninterpretedOption */
            uninterpretedOption?: (google.protobuf.IUninterpretedOption[]|null);
        }

        /** Represents a MethodOptions. */
        class MethodOptions implements IMethodOptions {

            /**
             * Constructs a new MethodOptions.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IMethodOptions);

            /** MethodOptions deprecated. */
            public deprecated: boolean;

            /** MethodOptions uninterpretedOption. */
            public uninterpretedOption: google.protobuf.IUninterpretedOption[];

            /**
             * Creates a new MethodOptions instance using the specified properties.
             * @param [properties] Properties to set
             * @returns MethodOptions instance
             */
            public static create(properties?: google.protobuf.IMethodOptions): google.protobuf.MethodOptions;

            /**
             * Encodes the specified MethodOptions message. Does not implicitly {@link google.protobuf.MethodOptions.verify|verify} messages.
             * @param message MethodOptions message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IMethodOptions, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified MethodOptions message, length delimited. Does not implicitly {@link google.protobuf.MethodOptions.verify|verify} messages.
             * @param message MethodOptions message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IMethodOptions, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a MethodOptions message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns MethodOptions
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.MethodOptions;

            /**
             * Decodes a MethodOptions message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns MethodOptions
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.MethodOptions;

            /**
             * Creates a MethodOptions message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns MethodOptions
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.MethodOptions;

            /**
             * Creates a plain object from a MethodOptions message. Also converts values to other types if specified.
             * @param message MethodOptions
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.MethodOptions, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this MethodOptions to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for MethodOptions
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }

        /** Properties of an UninterpretedOption. */
        interface IUninterpretedOption {

            /** UninterpretedOption name */
            name?: (google.protobuf.UninterpretedOption.INamePart[]|null);

            /** UninterpretedOption identifierValue */
            identifierValue?: (string|null);

            /** UninterpretedOption positiveIntValue */
            positiveIntValue?: (Long|null);

            /** UninterpretedOption negativeIntValue */
            negativeIntValue?: (Long|null);

            /** UninterpretedOption doubleValue */
            doubleValue?: (number|null);

            /** UninterpretedOption stringValue */
            stringValue?: (Uint8Array|null);

            /** UninterpretedOption aggregateValue */
            aggregateValue?: (string|null);
        }

        /** Represents an UninterpretedOption. */
        class UninterpretedOption implements IUninterpretedOption {

            /**
             * Constructs a new UninterpretedOption.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IUninterpretedOption);

            /** UninterpretedOption name. */
            public name: google.protobuf.UninterpretedOption.INamePart[];

            /** UninterpretedOption identifierValue. */
            public identifierValue: string;

            /** UninterpretedOption positiveIntValue. */
            public positiveIntValue: Long;

            /** UninterpretedOption negativeIntValue. */
            public negativeIntValue: Long;

            /** UninterpretedOption doubleValue. */
            public doubleValue: number;

            /** UninterpretedOption stringValue. */
            public stringValue: Uint8Array;

            /** UninterpretedOption aggregateValue. */
            public aggregateValue: string;

            /**
             * Creates a new UninterpretedOption instance using the specified properties.
             * @param [properties] Properties to set
             * @returns UninterpretedOption instance
             */
            public static create(properties?: google.protobuf.IUninterpretedOption): google.protobuf.UninterpretedOption;

            /**
             * Encodes the specified UninterpretedOption message. Does not implicitly {@link google.protobuf.UninterpretedOption.verify|verify} messages.
             * @param message UninterpretedOption message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IUninterpretedOption, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified UninterpretedOption message, length delimited. Does not implicitly {@link google.protobuf.UninterpretedOption.verify|verify} messages.
             * @param message UninterpretedOption message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IUninterpretedOption, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes an UninterpretedOption message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns UninterpretedOption
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.UninterpretedOption;

            /**
             * Decodes an UninterpretedOption message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns UninterpretedOption
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.UninterpretedOption;

            /**
             * Creates an UninterpretedOption message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns UninterpretedOption
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.UninterpretedOption;

            /**
             * Creates a plain object from an UninterpretedOption message. Also converts values to other types if specified.
             * @param message UninterpretedOption
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.UninterpretedOption, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this UninterpretedOption to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for UninterpretedOption
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }

        namespace UninterpretedOption {

            /** Properties of a NamePart. */
            interface INamePart {

                /** NamePart namePart */
                namePart: string;

                /** NamePart isExtension */
                isExtension: boolean;
            }

            /** Represents a NamePart. */
            class NamePart implements INamePart {

                /**
                 * Constructs a new NamePart.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: google.protobuf.UninterpretedOption.INamePart);

                /** NamePart namePart. */
                public namePart: string;

                /** NamePart isExtension. */
                public isExtension: boolean;

                /**
                 * Creates a new NamePart instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns NamePart instance
                 */
                public static create(properties?: google.protobuf.UninterpretedOption.INamePart): google.protobuf.UninterpretedOption.NamePart;

                /**
                 * Encodes the specified NamePart message. Does not implicitly {@link google.protobuf.UninterpretedOption.NamePart.verify|verify} messages.
                 * @param message NamePart message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: google.protobuf.UninterpretedOption.INamePart, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified NamePart message, length delimited. Does not implicitly {@link google.protobuf.UninterpretedOption.NamePart.verify|verify} messages.
                 * @param message NamePart message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: google.protobuf.UninterpretedOption.INamePart, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a NamePart message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns NamePart
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.UninterpretedOption.NamePart;

                /**
                 * Decodes a NamePart message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns NamePart
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.UninterpretedOption.NamePart;

                /**
                 * Creates a NamePart message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns NamePart
                 */
                public static fromObject(object: { [k: string]: any }): google.protobuf.UninterpretedOption.NamePart;

                /**
                 * Creates a plain object from a NamePart message. Also converts values to other types if specified.
                 * @param message NamePart
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: google.protobuf.UninterpretedOption.NamePart, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this NamePart to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for NamePart
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }
        }

        /** Properties of a SourceCodeInfo. */
        interface ISourceCodeInfo {

            /** SourceCodeInfo location */
            location?: (google.protobuf.SourceCodeInfo.ILocation[]|null);
        }

        /** Represents a SourceCodeInfo. */
        class SourceCodeInfo implements ISourceCodeInfo {

            /**
             * Constructs a new SourceCodeInfo.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.ISourceCodeInfo);

            /** SourceCodeInfo location. */
            public location: google.protobuf.SourceCodeInfo.ILocation[];

            /**
             * Creates a new SourceCodeInfo instance using the specified properties.
             * @param [properties] Properties to set
             * @returns SourceCodeInfo instance
             */
            public static create(properties?: google.protobuf.ISourceCodeInfo): google.protobuf.SourceCodeInfo;

            /**
             * Encodes the specified SourceCodeInfo message. Does not implicitly {@link google.protobuf.SourceCodeInfo.verify|verify} messages.
             * @param message SourceCodeInfo message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.ISourceCodeInfo, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified SourceCodeInfo message, length delimited. Does not implicitly {@link google.protobuf.SourceCodeInfo.verify|verify} messages.
             * @param message SourceCodeInfo message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.ISourceCodeInfo, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a SourceCodeInfo message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns SourceCodeInfo
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.SourceCodeInfo;

            /**
             * Decodes a SourceCodeInfo message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns SourceCodeInfo
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.SourceCodeInfo;

            /**
             * Creates a SourceCodeInfo message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns SourceCodeInfo
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.SourceCodeInfo;

            /**
             * Creates a plain object from a SourceCodeInfo message. Also converts values to other types if specified.
             * @param message SourceCodeInfo
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.SourceCodeInfo, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this SourceCodeInfo to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for SourceCodeInfo
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }

        namespace SourceCodeInfo {

            /** Properties of a Location. */
            interface ILocation {

                /** Location path */
                path?: (number[]|null);

                /** Location span */
                span?: (number[]|null);

                /** Location leadingComments */
                leadingComments?: (string|null);

                /** Location trailingComments */
                trailingComments?: (string|null);

                /** Location leadingDetachedComments */
                leadingDetachedComments?: (string[]|null);
            }

            /** Represents a Location. */
            class Location implements ILocation {

                /**
                 * Constructs a new Location.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: google.protobuf.SourceCodeInfo.ILocation);

                /** Location path. */
                public path: number[];

                /** Location span. */
                public span: number[];

                /** Location leadingComments. */
                public leadingComments: string;

                /** Location trailingComments. */
                public trailingComments: string;

                /** Location leadingDetachedComments. */
                public leadingDetachedComments: string[];

                /**
                 * Creates a new Location instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns Location instance
                 */
                public static create(properties?: google.protobuf.SourceCodeInfo.ILocation): google.protobuf.SourceCodeInfo.Location;

                /**
                 * Encodes the specified Location message. Does not implicitly {@link google.protobuf.SourceCodeInfo.Location.verify|verify} messages.
                 * @param message Location message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: google.protobuf.SourceCodeInfo.ILocation, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified Location message, length delimited. Does not implicitly {@link google.protobuf.SourceCodeInfo.Location.verify|verify} messages.
                 * @param message Location message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: google.protobuf.SourceCodeInfo.ILocation, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes a Location message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns Location
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.SourceCodeInfo.Location;

                /**
                 * Decodes a Location message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns Location
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.SourceCodeInfo.Location;

                /**
                 * Creates a Location message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns Location
                 */
                public static fromObject(object: { [k: string]: any }): google.protobuf.SourceCodeInfo.Location;

                /**
                 * Creates a plain object from a Location message. Also converts values to other types if specified.
                 * @param message Location
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: google.protobuf.SourceCodeInfo.Location, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this Location to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for Location
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }
        }

        /** Properties of a GeneratedCodeInfo. */
        interface IGeneratedCodeInfo {

            /** GeneratedCodeInfo annotation */
            annotation?: (google.protobuf.GeneratedCodeInfo.IAnnotation[]|null);
        }

        /** Represents a GeneratedCodeInfo. */
        class GeneratedCodeInfo implements IGeneratedCodeInfo {

            /**
             * Constructs a new GeneratedCodeInfo.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IGeneratedCodeInfo);

            /** GeneratedCodeInfo annotation. */
            public annotation: google.protobuf.GeneratedCodeInfo.IAnnotation[];

            /**
             * Creates a new GeneratedCodeInfo instance using the specified properties.
             * @param [properties] Properties to set
             * @returns GeneratedCodeInfo instance
             */
            public static create(properties?: google.protobuf.IGeneratedCodeInfo): google.protobuf.GeneratedCodeInfo;

            /**
             * Encodes the specified GeneratedCodeInfo message. Does not implicitly {@link google.protobuf.GeneratedCodeInfo.verify|verify} messages.
             * @param message GeneratedCodeInfo message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IGeneratedCodeInfo, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified GeneratedCodeInfo message, length delimited. Does not implicitly {@link google.protobuf.GeneratedCodeInfo.verify|verify} messages.
             * @param message GeneratedCodeInfo message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IGeneratedCodeInfo, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a GeneratedCodeInfo message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns GeneratedCodeInfo
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.GeneratedCodeInfo;

            /**
             * Decodes a GeneratedCodeInfo message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns GeneratedCodeInfo
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.GeneratedCodeInfo;

            /**
             * Creates a GeneratedCodeInfo message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns GeneratedCodeInfo
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.GeneratedCodeInfo;

            /**
             * Creates a plain object from a GeneratedCodeInfo message. Also converts values to other types if specified.
             * @param message GeneratedCodeInfo
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.GeneratedCodeInfo, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this GeneratedCodeInfo to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for GeneratedCodeInfo
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }

        namespace GeneratedCodeInfo {

            /** Properties of an Annotation. */
            interface IAnnotation {

                /** Annotation path */
                path?: (number[]|null);

                /** Annotation sourceFile */
                sourceFile?: (string|null);

                /** Annotation begin */
                begin?: (number|null);

                /** Annotation end */
                end?: (number|null);
            }

            /** Represents an Annotation. */
            class Annotation implements IAnnotation {

                /**
                 * Constructs a new Annotation.
                 * @param [properties] Properties to set
                 */
                constructor(properties?: google.protobuf.GeneratedCodeInfo.IAnnotation);

                /** Annotation path. */
                public path: number[];

                /** Annotation sourceFile. */
                public sourceFile: string;

                /** Annotation begin. */
                public begin: number;

                /** Annotation end. */
                public end: number;

                /**
                 * Creates a new Annotation instance using the specified properties.
                 * @param [properties] Properties to set
                 * @returns Annotation instance
                 */
                public static create(properties?: google.protobuf.GeneratedCodeInfo.IAnnotation): google.protobuf.GeneratedCodeInfo.Annotation;

                /**
                 * Encodes the specified Annotation message. Does not implicitly {@link google.protobuf.GeneratedCodeInfo.Annotation.verify|verify} messages.
                 * @param message Annotation message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encode(message: google.protobuf.GeneratedCodeInfo.IAnnotation, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Encodes the specified Annotation message, length delimited. Does not implicitly {@link google.protobuf.GeneratedCodeInfo.Annotation.verify|verify} messages.
                 * @param message Annotation message or plain object to encode
                 * @param [writer] Writer to encode to
                 * @returns Writer
                 */
                public static encodeDelimited(message: google.protobuf.GeneratedCodeInfo.IAnnotation, writer?: $protobuf.Writer): $protobuf.Writer;

                /**
                 * Decodes an Annotation message from the specified reader or buffer.
                 * @param reader Reader or buffer to decode from
                 * @param [length] Message length if known beforehand
                 * @returns Annotation
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.GeneratedCodeInfo.Annotation;

                /**
                 * Decodes an Annotation message from the specified reader or buffer, length delimited.
                 * @param reader Reader or buffer to decode from
                 * @returns Annotation
                 * @throws {Error} If the payload is not a reader or valid buffer
                 * @throws {$protobuf.util.ProtocolError} If required fields are missing
                 */
                public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.GeneratedCodeInfo.Annotation;

                /**
                 * Creates an Annotation message from a plain object. Also converts values to their respective internal types.
                 * @param object Plain object
                 * @returns Annotation
                 */
                public static fromObject(object: { [k: string]: any }): google.protobuf.GeneratedCodeInfo.Annotation;

                /**
                 * Creates a plain object from an Annotation message. Also converts values to other types if specified.
                 * @param message Annotation
                 * @param [options] Conversion options
                 * @returns Plain object
                 */
                public static toObject(message: google.protobuf.GeneratedCodeInfo.Annotation, options?: $protobuf.IConversionOptions): { [k: string]: any };

                /**
                 * Converts this Annotation to JSON.
                 * @returns JSON object
                 */
                public toJSON(): { [k: string]: any };

                /**
                 * Gets the default type url for Annotation
                 * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
                 * @returns The default type url
                 */
                public static getTypeUrl(typeUrlPrefix?: string): string;
            }
        }

        /** Properties of a Duration. */
        interface IDuration {

            /** Duration seconds */
            seconds?: (Long|null);

            /** Duration nanos */
            nanos?: (number|null);
        }

        /** Represents a Duration. */
        class Duration implements IDuration {

            /**
             * Constructs a new Duration.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IDuration);

            /** Duration seconds. */
            public seconds: Long;

            /** Duration nanos. */
            public nanos: number;

            /**
             * Creates a new Duration instance using the specified properties.
             * @param [properties] Properties to set
             * @returns Duration instance
             */
            public static create(properties?: google.protobuf.IDuration): google.protobuf.Duration;

            /**
             * Encodes the specified Duration message. Does not implicitly {@link google.protobuf.Duration.verify|verify} messages.
             * @param message Duration message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IDuration, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified Duration message, length delimited. Does not implicitly {@link google.protobuf.Duration.verify|verify} messages.
             * @param message Duration message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IDuration, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes a Duration message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns Duration
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.Duration;

            /**
             * Decodes a Duration message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns Duration
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.Duration;

            /**
             * Creates a Duration message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns Duration
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.Duration;

            /**
             * Creates a plain object from a Duration message. Also converts values to other types if specified.
             * @param message Duration
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.Duration, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this Duration to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for Duration
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }

        /** Properties of an Empty. */
        interface IEmpty {
        }

        /** Represents an Empty. */
        class Empty implements IEmpty {

            /**
             * Constructs a new Empty.
             * @param [properties] Properties to set
             */
            constructor(properties?: google.protobuf.IEmpty);

            /**
             * Creates a new Empty instance using the specified properties.
             * @param [properties] Properties to set
             * @returns Empty instance
             */
            public static create(properties?: google.protobuf.IEmpty): google.protobuf.Empty;

            /**
             * Encodes the specified Empty message. Does not implicitly {@link google.protobuf.Empty.verify|verify} messages.
             * @param message Empty message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encode(message: google.protobuf.IEmpty, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified Empty message, length delimited. Does not implicitly {@link google.protobuf.Empty.verify|verify} messages.
             * @param message Empty message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            public static encodeDelimited(message: google.protobuf.IEmpty, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes an Empty message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns Empty
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): google.protobuf.Empty;

            /**
             * Decodes an Empty message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns Empty
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): google.protobuf.Empty;

            /**
             * Creates an Empty message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns Empty
             */
            public static fromObject(object: { [k: string]: any }): google.protobuf.Empty;

            /**
             * Creates a plain object from an Empty message. Also converts values to other types if specified.
             * @param message Empty
             * @param [options] Conversion options
             * @returns Plain object
             */
            public static toObject(message: google.protobuf.Empty, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this Empty to JSON.
             * @returns JSON object
             */
            public toJSON(): { [k: string]: any };

            /**
             * Gets the default type url for Empty
             * @param [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns The default type url
             */
            public static getTypeUrl(typeUrlPrefix?: string): string;
        }
    }
}
