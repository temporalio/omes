import * as path from 'node:path';
import * as grpc from '@grpc/grpc-js';
import * as protoLoader from '@grpc/proto-loader';
import type { ProtoGrpcType } from './generated/api';
import type {
  ProjectServiceClient,
  ProjectServiceDefinition,
} from './generated/temporal/omes/projects/v1/ProjectService';

export type ProjectServiceClientConstructor = new (
  address: string,
  credentials: grpc.ChannelCredentials,
  options?: Partial<grpc.ClientOptions>,
) => ProjectServiceClient;

// This harness is repo-internal, so the runtime can load the source proto directly.
const packageDefinition = protoLoader.loadSync(
  path.resolve(__dirname, '../../../../proto/harness/api/api.proto'),
  {
    longs: Number,
  },
);
const proto = grpc.loadPackageDefinition(packageDefinition) as unknown as ProtoGrpcType;

export const projectServiceDefinition: ProjectServiceDefinition =
  proto.temporal.omes.projects.v1.ProjectService.service;
export const projectServiceClientConstructor = proto.temporal.omes.projects.v1
  .ProjectService as unknown as ProjectServiceClientConstructor;
