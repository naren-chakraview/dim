// Route configuration types (generated from JSON schema)

export interface RouteConfig {
  version: number;
  domain?: string;
  imports?: string[];
  sources: Record<string, SourceConfig>;
  routes: Record<string, Route>;
  sinks: Record<string, SinkConfig>;
  contracts?: Record<string, Contract>;
  error_path?: ErrorPath;
}

export interface SourceConfig {
  type: 'http' | 'file' | 'kafka' | 'amqp' | 'sftp';
  [key: string]: any;
}

export interface Route {
  from: string;
  steps?: Step[];
  to?: string;
  route_version?: string;
}

export interface Step {
  [key: string]: StepConfig;
}

export interface StepConfig {
  [key: string]: any;
}

export interface SinkConfig {
  type: 'file' | 'http' | 'kafka' | 's3';
  [key: string]: any;
}

export interface Contract {
  schema?: string | object;
  [key: string]: any;
}

export interface ErrorPath {
  steps?: Step[];
  sinks?: Record<string, SinkConfig>;
}

// DAG visualization types

export interface DAGNode {
  id: string;
  label: string;
  type: 'source' | 'step' | 'sink' | 'error_sink';
  position: { x: number; y: number };
  data: {
    label: string;
    config?: any;
    icon?: string;
  };
}

export interface DAGEdge {
  id: string;
  source: string;
  target: string;
  animated?: boolean;
}

export interface DAGLayout {
  nodes: DAGNode[];
  edges: DAGEdge[];
}

// Step type to icon mapping
export const STEP_ICONS: Record<string, string> = {
  translate: '🔄',
  log: '📝',
  delay: '⏱️',
  http: '🌐',
  kafka: '📡',
  filter: '🔍',
  split: '🔀',
  merge: '🔗',
  transform: '✨',
  map: '🗺️',
  reduce: '📉',
  aggregate: '📊',
  deduplicate: '🔖',
  enrich: '➕',
  validate: '✅',
  enforce: '🛡️',
};

export const ADAPTER_ICONS: Record<string, string> = {
  http: '🌐',
  file: '📄',
  kafka: '📡',
  amqp: '🐰',
  sftp: '🔒',
  s3: '☁️',
  database: '🗄️',
};
