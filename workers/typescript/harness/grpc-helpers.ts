import * as path from 'node:path';
import * as grpc from '@grpc/grpc-js';
import * as protoLoader from '@grpc/proto-loader';
import type { ProtoGrpcType } from './api/api';
import type {
  ProjectServiceClient,
  ProjectServiceDefinition,
} from './api/temporal/omes/projects/v1/ProjectService';

export type ProjectServiceClientConstructor = new (
  address: string,
  credentials: grpc.ChannelCredentials,
  options?: Partial<grpc.ClientOptions>,
) => ProjectServiceClient;

// Packaged consumers (e.g. temp worker builds in cmd/dev test) do not have repo-relative
// access to workers/proto, so built outputs must carry api.proto next to the
// compiled harness package.
const packageDefinition = protoLoader.loadSync(path.resolve(__dirname, './api/api.proto'), {
  longs: Number,
});
const proto = grpc.loadPackageDefinition(packageDefinition) as unknown as ProtoGrpcType;

export const projectServiceDefinition: ProjectServiceDefinition =
  proto.temporal.omes.projects.v1.ProjectService.service;
export const projectServiceClientConstructor = proto.temporal.omes.projects.v1
  .ProjectService as unknown as ProjectServiceClientConstructor;
