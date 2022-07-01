// import process from 'process';
interface NucleusApi {
  deploy(serv: NucleusService): Promise<NucleusServiceResponse>;

  listServices(): Promise<string[]>;
  getService(serviceName: string): Promise<NucleusServiceResponse>;

  logWindow(serviceName: string, opts?: LogWindowOpts): Promise<string[]>;
  tailLogs(serviceName: string, tail?: boolean): ReadableStream;
}

interface LogWindowOpts {
  window: '15m'| '1h' | '1d'
}

// deploy(new NucleusService())
// deploy(new NucleusDatabase())

interface NucleusService {
  name: string;
  buildCommand?: string;
  startCommand?: string;
  isPrivateService?: boolean;
  envVars?: Record<string, string>;
  secrets?: SecretsApi;
  artifact: Artifact;
}

interface SecretsApi {}

type Artifact = FileContentsArtifact | DockerImageArtifact;

interface FileContentsArtifact {
  runtime: 'nodejs' | 'python' | 'go';
  directory: string;
}

interface DockerImageArtifact {
  image: string;
  creds: any;
}

interface NucleusServiceResponse {
  url: string;
  externalUrl: string;
  internalUrl: string;
}

interface NucleusSecrets {
  type: 'nucleus';
  value: Record<string, string>; // encrypted
}

interface CMKSecrets {
  type: 'aws-cmk';
  kmsId: string;
  iamRoleArn: string;
  accountId: string;
}

interface VaultSecrets {
  type: 'hashi-vault';
  //
}

(async () => {
const nucleusClient: NucleusApi = {
  clientId: "CLIENT_ID",
  clientSecret: "CLIENT_SECRET",
} as any;



// const fooPrivServiceResult = await nucleusClient.deploy({
//   name: 'foo-service',
//   artifact: {
//     runtime: 'nodejs',
//     directory: '.',
//   },
//   buildCommand: 'npm build',
//   startCommand: 'npm start',
// });

const fooPrivServiceResult = await nucleusClient.getService('foo-service');

const pubServ = await nucleusClient.deploy({
  name: 'foo-service',
  envVars: {
    FOO_URL: fooPrivServiceResult.url,
  },
  artifact: {
    image: 'hello-world',
    creds: {},
  }
})

})();




