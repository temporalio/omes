import * as fs from 'node:fs';
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

function harnessApiProtoPath(): string {
  const candidates = [
    path.resolve(__dirname, './api/api.proto'),
    path.resolve(__dirname, '../../harness/api/api.proto'),
    path.resolve(__dirname, '../../../harness/api/api.proto'),
  ];
  const protoPath = candidates.find((candidate) => fs.existsSync(candidate));
  if (protoPath === undefined) {
    throw new Error(`Could not find harness api.proto in: ${candidates.join(', ')}`);
  }
  return protoPath;
}

const packageDefinition = protoLoader.loadSync(harnessApiProtoPath(), {
  longs: Number,
});
const proto = grpc.loadPackageDefinition(packageDefinition) as unknown as ProtoGrpcType;

export const projectServiceDefinition: ProjectServiceDefinition =
  proto.temporal.omes.projects.v1.ProjectService.service;
export const projectServiceClientConstructor = proto.temporal.omes.projects.v1
  .ProjectService as unknown as ProjectServiceClientConstructor;
